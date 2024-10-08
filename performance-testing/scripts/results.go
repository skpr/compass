package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	FailLimit    = 5
	RequestLimit = 200
)

type Report struct {
	RootGroup struct {
		Checks struct {
			OK struct {
				Fails float32 `json:"fails"`
			} `json:"ok"`
		} `json:"checks"`
	} `json:"root_group"`
	Metrics struct {
		HTTPReqDuration struct {
			Avg float32 `json:"avg"`
		} `json:"http_req_duration"`
	} `json:"metrics"`
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	control, err := getReport("control/report.json")
	if err != nil {
		return fmt.Errorf("failed to load control report: %w", err)
	}

	if control.RootGroup.Checks.OK.Fails > FailLimit {
		return fmt.Errorf("control errors (%d) are over the limit (%d)", int(control.RootGroup.Checks.OK.Fails), FailLimit)
	}

	installed, err := getReport("installed/report.json")
	if err != nil {
		return fmt.Errorf("failed to load installed report: %w", err)
	}

	if installed.RootGroup.Checks.OK.Fails > FailLimit {
		return fmt.Errorf("installed errors (%d) are over the limit (%d)", int(installed.RootGroup.Checks.OK.Fails), FailLimit)
	}

	enabled, err := getReport("enabled/report.json")
	if err != nil {
		return fmt.Errorf("failed to load enabled report: %w", err)
	}

	if enabled.RootGroup.Checks.OK.Fails > FailLimit {
		return fmt.Errorf("enabled errors (%d) are over the limit (%d)", int(enabled.RootGroup.Checks.OK.Fails), FailLimit)
	}

	collector, err := getReport("collector/report.json")
	if err != nil {
		return fmt.Errorf("failed to load collector report: %w", err)
	}

	if collector.RootGroup.Checks.OK.Fails > FailLimit {
		return fmt.Errorf("collector errors (%d) are over the limit (%d)", int(collector.RootGroup.Checks.OK.Fails), FailLimit)
	}

	var errs []error

	disabledDiff := installed.Metrics.HTTPReqDuration.Avg - control.Metrics.HTTPReqDuration.Avg
	if disabledDiff > RequestLimit {
		errs = append(errs, fmt.Errorf("extension report exceeded the request limit"))
	}

	enabledDiff := enabled.Metrics.HTTPReqDuration.Avg - control.Metrics.HTTPReqDuration.Avg
	if enabledDiff > RequestLimit {
		errs = append(errs, fmt.Errorf("extension report exceeded the request limit"))
	}

	collectorDiff := collector.Metrics.HTTPReqDuration.Avg - control.Metrics.HTTPReqDuration.Avg
	if collectorDiff > RequestLimit {
		errs = append(errs, fmt.Errorf("collector report exceeded the request limit"))
	}

	summaryTemplate := `| Test      | Average | Diff (Compared to Control) |
|-----------|---------|----------------------------|
| Control   | %dms    |                            |
| Installed | %dms    | %dms                       |
| Enabled   | %dms    | %dms                       |
| Collector | %dms    | %dms                       |`

	summary := fmt.Sprintf(summaryTemplate,
		int(control.Metrics.HTTPReqDuration.Avg),
		int(installed.Metrics.HTTPReqDuration.Avg),
		int(disabledDiff),
		int(enabled.Metrics.HTTPReqDuration.Avg),
		int(enabledDiff),
		int(collector.Metrics.HTTPReqDuration.Avg),
		int(collectorDiff),
	)

	data := []byte(summary)

	err = os.WriteFile("summary.md", data, 0644)
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func getReport(path string) (Report, error) {
	var report Report

	file, err := os.Open(path)
	if err != nil {
		return report, fmt.Errorf("failed to open file: %w", err)
	}

	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return report, fmt.Errorf("failed to read data: %w", err)
	}

	if err := json.Unmarshal(data, &report); err != nil {
		return report, fmt.Errorf("failed to unmarshal json data: %w", err)
	}

	return report, nil
}
