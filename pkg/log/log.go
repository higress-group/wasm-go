// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

type Log interface {
	Trace(msg string)
	Tracef(format string, args ...interface{})
	Debug(msg string)
	Debugf(format string, args ...interface{})
	Info(msg string)
	Infof(format string, args ...interface{})
	Warn(msg string)
	Warnf(format string, args ...interface{})
	Error(msg string)
	Errorf(format string, args ...interface{})
	Critical(msg string)
	Criticalf(format string, args ...interface{})
	ResetID(pluginID string)
}

var pluginLog Log

// safeLogEnabled controls whether sensitive logs should be suppressed.
// When enabled, UnsafeInfo/UnsafeInfof calls will be silently ignored.
var safeLogEnabled bool

func SetPluginLog(log Log) {
	pluginLog = log
}

// SetSafeLogEnabled enables or disables safe log mode.
// When safe log mode is enabled, sensitive logs (printed via UnsafeInfo/UnsafeInfof)
// will be suppressed to prevent leaking sensitive information like headers and body content.
func SetSafeLogEnabled(enabled bool) {
	safeLogEnabled = enabled
}

// IsSafeLogEnabled returns whether safe log mode is currently enabled.
func IsSafeLogEnabled() bool {
	return safeLogEnabled
}

func Trace(msg string) {
	pluginLog.Trace(msg)
}

func Tracef(format string, args ...interface{}) {
	pluginLog.Tracef(format, args...)
}

func Debug(msg string) {
	pluginLog.Debug(msg)
}

func Debugf(format string, args ...interface{}) {
	pluginLog.Debugf(format, args...)
}

func Info(msg string) {
	pluginLog.Info(msg)
}

func Infof(format string, args ...interface{}) {
	pluginLog.Infof(format, args...)
}

func Warn(msg string) {
	pluginLog.Warn(msg)
}

func Warnf(format string, args ...interface{}) {
	pluginLog.Warnf(format, args...)
}

func Error(msg string) {
	pluginLog.Error(msg)
}

func Errorf(format string, args ...interface{}) {
	pluginLog.Errorf(format, args...)
}

func Critical(msg string) {
	pluginLog.Critical(msg)
}

func Criticalf(format string, args ...interface{}) {
	pluginLog.Criticalf(format, args...)
}

// UnsafeInfo logs a message at Info level only if safe log mode is disabled.
// Use this for sensitive information that should not be logged in production.
func UnsafeInfo(msg string) {
	if !safeLogEnabled {
		pluginLog.Info(msg)
	}
}

// UnsafeInfof logs a formatted message at Info level only if safe log mode is disabled.
// Use this for sensitive information that should not be logged in production.
func UnsafeInfof(format string, args ...interface{}) {
	if !safeLogEnabled {
		pluginLog.Infof(format, args...)
	}
}

// UnsafeDebug logs a message at Debug level only if safe log mode is disabled.
// Use this for sensitive information that should not be logged in production.
func UnsafeDebug(msg string) {
	if !safeLogEnabled {
		pluginLog.Debug(msg)
	}
}

// UnsafeDebugf logs a formatted message at Debug level only if safe log mode is disabled.
// Use this for sensitive information that should not be logged in production.
func UnsafeDebugf(format string, args ...interface{}) {
	if !safeLogEnabled {
		pluginLog.Debugf(format, args...)
	}
}
