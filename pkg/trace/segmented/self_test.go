package segmented

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
)

func call(name string, start, elapsed int64) trace.FunctionCall {
	return trace.FunctionCall{Name: name, StartTime: start, Elapsed: elapsed}
}

func TestSelfTime_NoCalls(t *testing.T) {
	assert.Empty(t, SelfTime(nil))
}

func TestSelfTime_SingleCall(t *testing.T) {
	assert.Equal(t, []int64{100}, SelfTime([]trace.FunctionCall{call("a", 0, 100)}))
}

func TestSelfTime_Siblings(t *testing.T) {
	// Two calls which do not nest each keep all of their time.
	self := SelfTime([]trace.FunctionCall{
		call("a", 0, 30),
		call("b", 40, 20),
	})

	assert.Equal(t, []int64{30, 20}, self)
}

func TestSelfTime_ParentAndChild(t *testing.T) {
	self := SelfTime([]trace.FunctionCall{
		call("parent", 0, 100),
		call("child", 10, 60),
	})

	assert.Equal(t, []int64{40, 60}, self)
}

func TestSelfTime_OnlyDirectChildrenAreDeducted(t *testing.T) {
	// The grandchild's time is already inside the child's, so deducting it from
	// the parent as well would count it twice.
	self := SelfTime([]trace.FunctionCall{
		call("parent", 0, 100),
		call("child", 10, 80),
		call("grandchild", 20, 50),
	})

	assert.Equal(t, []int64{20, 30, 50}, self)
}

func TestSelfTime_MultipleChildren(t *testing.T) {
	self := SelfTime([]trace.FunctionCall{
		call("parent", 0, 100),
		call("first", 0, 30),
		call("second", 40, 40),
	})

	assert.Equal(t, []int64{30, 30, 40}, self)
}

// A parent which starts at the same instant as its child has to be recognised
// as the parent, which is what the length tiebreak is for.
func TestSelfTime_SharedStart(t *testing.T) {
	self := SelfTime([]trace.FunctionCall{
		call("child", 0, 40),
		call("parent", 0, 100),
	})

	assert.Equal(t, []int64{40, 60}, self)
}

func TestSelfTime_OrderOfInputDoesNotMatter(t *testing.T) {
	forwards := SelfTime([]trace.FunctionCall{
		call("parent", 0, 100),
		call("child", 10, 60),
	})

	backwards := SelfTime([]trace.FunctionCall{
		call("child", 10, 60),
		call("parent", 0, 100),
	})

	assert.Equal(t, forwards[0], backwards[1])
	assert.Equal(t, forwards[1], backwards[0])
}

// The call tree is incomplete, because the probe only fires above a threshold,
// so intervals can overhang their parent. Self time must never go negative.
func TestSelfTime_NeverNegative(t *testing.T) {
	self := SelfTime([]trace.FunctionCall{
		call("parent", 0, 50),
		call("overhanging child", 10, 500),
	})

	for i, value := range self {
		assert.GreaterOrEqual(t, value, int64(0), "call %d", i)
	}
}

// The case from the screenshot in the repository, which is what motivated all
// of this: a stack of framework frames that each wrap the whole request, and
// one function underneath doing the actual work. Ranking by elapsed puts the
// framework on top; ranking by self time finds the hotspot.
func TestSelfTime_FindsTheHotspotUnderTheFramework(t *testing.T) {
	calls := []trace.FunctionCall{
		call(`Drupal\Component\EventDispatcher\ContainerAwareEventDispatcher::dispatch`, 0, 6293),
		call(`Symfony\Component\HttpKernel\HttpKernel::terminate`, 0, 6293),
		call(`Drupal\Core\Cron::run`, 1, 6284),
		call(`Drupal\Core\Cron::invokeCronHandlers`, 2, 6282),
		call("search_cron", 3, 4379),
		call(`Drupal\help\Plugin\Search\HelpSearch::updateIndex`, 4, 3047),
	}

	self := SelfTime(calls)

	hottest := 0
	for i := range self {
		if self[i] > self[hottest] {
			hottest = i
		}
	}

	require.Equal(t, `Drupal\help\Plugin\Search\HelpSearch::updateIndex`, calls[hottest].Name,
		"self time should rank the hotspot first, not the frames wrapping it")

	// And the frames which merely wrap it are nearly free.
	assert.Less(t, self[0], int64(10))
	assert.Less(t, self[1], int64(10))
}

func TestUnmarshal_CarriesSelfTime(t *testing.T) {
	segmentedTrace := Unmarshal(trace.Trace{
		Metadata: trace.Metadata{StartTime: 0, EndTime: 1000},
		FunctionCalls: []trace.FunctionCall{
			call("parent", 0, 1000),
			call("child", 100, 600),
		},
	}, 50)

	byName := make(map[string]Span)
	for _, span := range segmentedTrace.Spans {
		byName[span.Name] = span
	}

	assert.Equal(t, int64(400), byName["parent"].SelfTime)
	assert.Equal(t, int64(600), byName["child"].SelfTime)
	assert.InDelta(t, 0.6, byName["child"].SelfShare(1000), 0.001)
}

// Repeated calls to the same function aggregate into one span when they land
// in the same segment, and their self time adds up rather than one winning.
func TestUnmarshal_AggregatesSelfTime(t *testing.T) {
	// A thousand nanoseconds over fifty segments is twenty per segment, so both
	// of these fall in the first one.
	segmentedTrace := Unmarshal(trace.Trace{
		Metadata: trace.Metadata{StartTime: 0, EndTime: 1000},
		FunctionCalls: []trace.FunctionCall{
			call("repeated", 0, 10),
			call("repeated", 11, 10),
		},
	}, 50)

	require.Len(t, segmentedTrace.Spans, 1)
	assert.Equal(t, int64(20), segmentedTrace.Spans[0].SelfTime)
	assert.Equal(t, 2, segmentedTrace.Spans[0].TotalFunctionCalls)
}

// A request too short to have one nanosecond per segment used to divide by
// zero here.
func TestUnmarshal_VeryShortTrace(t *testing.T) {
	assert.NotPanics(t, func() {
		Unmarshal(trace.Trace{
			Metadata:      trace.Metadata{StartTime: 0, EndTime: 5},
			FunctionCalls: []trace.FunctionCall{call("a", 0, 5)},
		}, 50)
	})
}

func TestUnmarshal_ZeroDurationTrace(t *testing.T) {
	assert.NotPanics(t, func() {
		Unmarshal(trace.Trace{
			Metadata:      trace.Metadata{StartTime: 100, EndTime: 100},
			FunctionCalls: []trace.FunctionCall{call("a", 100, 0)},
		}, 50)
	})
}
