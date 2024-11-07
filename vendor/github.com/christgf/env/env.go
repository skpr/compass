// Package env can retrieve and parse environment variables, with defaults.
package env

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// VarSet represents a set of environment variables managed as key-value pairs.
type VarSet struct {
	prefix string
}

// SetPrefix makes this VarSet prepend the value of prefix to every key before it
// looks it up in the environment using String, StringVar, Bool, Int, et al. Use
// the empty string to reset.
func (vs *VarSet) SetPrefix(prefix string) {
	vs.prefix = prefix
}

// Prefix returns the prefix for this VarSet, if any.
func (vs *VarSet) Prefix() string {
	return vs.prefix
}

// lookup applies the prefix for this VarSet to the key provided and attempts to
// retrieve the value of the corresponding environment variable. If the variable
// is present in the environment the value is returned and the boolean is true.
// Otherwise, the returned value will be empty and the boolean will be false.
func (vs *VarSet) lookup(key string) (string, bool) {
	if len(vs.prefix) > 0 {
		key = fmt.Sprintf("%s%s", vs.prefix, key)
	}

	return os.LookupEnv(key)
}

// Lookup retrieves the value of the environment variable named by the key. If
// the variable is present in the environment the value is returned and the
// boolean is true. Otherwise, the returned value will be empty and the boolean
// will be false.
func (vs *VarSet) Lookup(key string) (string, bool) {
	return vs.lookup(key)
}

// String retrieves the value of the environment variable named by the key. If
// the variable is present in the environment, its value (which may be empty) is
// returned, otherwise fallback is returned.
func (vs *VarSet) String(key string, fallback string) string {
	value, ok := vs.lookup(key)
	if !ok {
		return fallback
	}

	return value
}

// Bool retrieves the value of the environment variable named by the key, parses
// the value as a boolean, and returns the result. If the variable is not
// present or its value cannot be parsed, fallback is returned.
func (vs *VarSet) Bool(key string, fallback bool) bool {
	value, ok := vs.lookup(key)
	if !ok {
		return fallback
	}

	res, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return res
}

// Int retrieves the value of the environment variable named by the key, parses
// the value as an integer, and returns the result. If the variable is not
// present or its value cannot be parsed, fallback is returned.
func (vs *VarSet) Int(key string, fallback int) int {
	value, ok := vs.lookup(key)
	if !ok {
		return fallback
	}

	res, err := strconv.ParseInt(value, 10, strconv.IntSize)
	if err != nil {
		return fallback
	}

	return int(res)
}

// Int64 retrieves the value of the environment variable named by the key, parses
// the value as a 64-bit integer, and returns the result. If the variable is not
// present or its value cannot be parsed, fallback is returned.
func (vs *VarSet) Int64(key string, fallback int64) int64 {
	value, ok := vs.lookup(key)
	if !ok {
		return fallback
	}

	res, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}

	return res
}

// Uint retrieves the value of the environment variable named by the key, parses
// the value as an unsigned integer, and returns the result. If the variable is
// not present or its value cannot be parsed, fallback is returned.
func (vs *VarSet) Uint(key string, fallback uint) uint {
	value, ok := vs.lookup(key)
	if !ok {
		return fallback
	}

	res, err := strconv.ParseUint(value, 10, strconv.IntSize)
	if err != nil {
		return fallback
	}

	return uint(res)
}

// Uint64 retrieves the value of the environment variable named by the key,
// parses the value as an unsigned 64-bit integer, and returns the result. If the
// variable is not present or its value cannot be parsed, fallback is returned.
func (vs *VarSet) Uint64(key string, fallback uint64) uint64 {
	value, ok := vs.lookup(key)
	if !ok {
		return fallback
	}

	res, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fallback
	}

	return res
}

// Float32 retrieves the value of the environment variable named by the key,
// parses the value as a floating-point number, and returns the result. If the
// variable is not present or its value cannot be parsed, fallback is returned.
func (vs *VarSet) Float32(key string, fallback float32) float32 {
	value, ok := vs.lookup(key)
	if !ok {
		return fallback
	}

	res, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return fallback
	}

	return float32(res)
}

// Float64 retrieves the value of the environment variable named by the key,
// parses the value as a 64-bit floating-point number, and returns the result. If
// the variable is not present or its value cannot be parsed, fallback is
// returned.
func (vs *VarSet) Float64(key string, fallback float64) float64 {
	value, ok := vs.lookup(key)
	if !ok {
		return fallback
	}

	res, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}

	return res
}

// Duration retrieves the value of the environment variable named by the key,
// parses the value as time.Duration, and returns the result. If the variable is
// not present or its value cannot be parsed, fallback is returned.
func (vs *VarSet) Duration(key string, fallback time.Duration) time.Duration {
	value, ok := vs.lookup(key)
	if !ok {
		return fallback
	}

	res, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return res
}

// osVarSet is the default VarSet. Top-level functions such as String, StringVar,
// Bool, etc. are wrappers for the methods of osVarSet.
var osVarSet = &VarSet{prefix: ""}

// SetPrefix makes this VarSet prepend the value of prefix to every key before it
// looks it up in the environment using String, StringVar, Bool, Int, et al. Use
// the empty string to reset.
func SetPrefix(prefix string) {
	osVarSet.SetPrefix(prefix)
}

// Prefix returns the prefix for the default VarSet.
func Prefix() string {
	return osVarSet.Prefix()
}

// Lookup retrieves the value of the environment variable named by the key. If
// the variable is present in the environment the value is returned and the
// boolean is true. Otherwise, the returned value will be empty and the boolean
// will be false.
func Lookup(key string) (string, bool) {
	return osVarSet.Lookup(key)
}

// String retrieves the value of the environment variable named by the key. If
// the variable is present in the environment, its value (which may be empty) is
// returned, otherwise fallback is returned.
func String(key string, fallback string) string {
	return osVarSet.String(key, fallback)
}

// Bool retrieves the value of the environment variable named by the key, parses
// the value as a boolean, and returns the result. If the variable is not
// present or its value cannot be parsed, fallback is returned.
func Bool(key string, fallback bool) bool {
	return osVarSet.Bool(key, fallback)
}

// Int retrieves the value of the environment variable named by the key, parses
// the value as an integer, and returns the result. If the variable is not
// present or its value cannot be parsed, fallback is returned.
func Int(key string, fallback int) int {
	return osVarSet.Int(key, fallback)
}

// Int64 retrieves the value of the environment variable named by the key, parses
// the value as a 64-bit integer, and returns the result. If the variable is not
// present or its value cannot be parsed, fallback is returned.
func Int64(key string, fallback int64) int64 {
	return osVarSet.Int64(key, fallback)
}

// Uint retrieves the value of the environment variable named by the key, parses
// the value as an unsigned integer, and returns the result. If the variable is
// not present or its value cannot be parsed, fallback is returned.
func Uint(key string, fallback uint) uint {
	return osVarSet.Uint(key, fallback)
}

// Uint64 retrieves the value of the environment variable named by the key,
// parses the value as an unsigned 64-bit integer, and returns the result. If the
// variable is not present or its value cannot be parsed, fallback is returned.
func Uint64(key string, fallback uint64) uint64 {
	return osVarSet.Uint64(key, fallback)
}

// Float32 retrieves the value of the environment variable named by the key,
// parses the value as a floating-point number, and returns the result. If the
// variable is not present or its value cannot be parsed, fallback is returned.
func Float32(key string, fallback float32) float32 {
	return osVarSet.Float32(key, fallback)
}

// Float64 retrieves the value of the environment variable named by the key,
// parses the value as a 64-bit floating-point number, and returns the result. If
// the variable is not present or its value cannot be parsed, fallback is
// returned.
func Float64(key string, fallback float64) float64 {
	return osVarSet.Float64(key, fallback)
}

// Duration retrieves the value of the environment variable named by the key,
// parses the value as time.Duration, and returns the result. If the variable is
// not present or its value cannot be parsed, fallback is returned.
func Duration(key string, fallback time.Duration) time.Duration {
	return osVarSet.Duration(key, fallback)
}

// StringVar retrieves the value of the environment variable named by the key,
// and stores the result into the variable pointed by p.
func StringVar(p *string, key string, fallback string) {
	*p = osVarSet.String(key, fallback)
}

// BoolVar retrieves the value of the environment variable named by the key,
// parses the value as a boolean, and stores the result into the variable pointed
// by p.
func BoolVar(p *bool, key string, fallback bool) {
	*p = osVarSet.Bool(key, fallback)
}

// IntVar retrieves the value of the environment variable named by the key,
// parses the value as an integer, and stores the result into the variable
// pointed by p.
func IntVar(p *int, key string, fallback int) {
	*p = osVarSet.Int(key, fallback)
}

// Int64Var retrieves the value of the environment variable named by the key,
// parses the value as a 64-bit integer, and stores the result into the variable
// pointed by p.
func Int64Var(p *int64, key string, fallback int64) {
	*p = osVarSet.Int64(key, fallback)
}

// UintVar retrieves the value of the environment variable named by the key,
// parses the value as an unsigned integer, and stores the result into the
// variable pointed by p.
func UintVar(p *uint, key string, fallback uint) {
	*p = osVarSet.Uint(key, fallback)
}

// Uint64Var retrieves the value of the environment variable named by the key,
// parses the value as an unsigned 64-bit integer, and stores the result into the
// variable pointed by p.
func Uint64Var(p *uint64, key string, fallback uint64) {
	*p = osVarSet.Uint64(key, fallback)
}

// Float32Var retrieves the value of the environment variable named by the key,
// parses the value as a floating-point number, and stores the result into the
// variable pointed by p.
func Float32Var(p *float32, key string, fallback float32) {
	*p = osVarSet.Float32(key, fallback)
}

// Float64Var retrieves the value of the environment variable named by the key,
// parses the value as a 64-bit floating-point number, and stores the result into
// the variable pointed by p.
func Float64Var(p *float64, key string, fallback float64) {
	*p = osVarSet.Float64(key, fallback)
}

// DurationVar retrieves the value of the environment variable named by the key,
// parses the value as time.Duration, and stores the result into the variable
// pointed by p.
func DurationVar(p *time.Duration, key string, fallback time.Duration) {
	*p = osVarSet.Duration(key, fallback)
}
