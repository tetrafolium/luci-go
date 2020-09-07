// Copyright 2015 The LUCI Authors.
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

// Package gaeconfig implements LUCI-config service bindings backed by AppEngine
// storage and caching.
//
// Importing this module registers ${appid} and ${config_service_appid} config
// placeholder variables, see github.com/tetrafolium/luci-go/config/vars.
//
// DEPRECATED!
//
// Do not use outside of GAEv1. Use github.com/tetrafolium/luci-go/config/server/cfgmodule
// on GAEv2 and GKE instead.
package gaeconfig
