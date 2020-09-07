# Copyright 2019 The LUCI Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Exports proto modules with messages used in generated LUCI configs.

Prefer using this module over loading "@proto//..." modules directly. Proto
paths may change in a backward incompatible way. Using this module gives more
stability.
"""

load("@stdlib//internal/luci/descpb.star", "lucitypes_descpb")

lucitypes_descpb.register()

load("@proto//github.com/tetrafolium/luci-go/buildbucket/proto/common.proto", _common_pb = "buildbucket.v2")
load("@proto//github.com/tetrafolium/luci-go/buildbucket/proto/project_config.proto", _buildbucket_pb = "buildbucket")
load("@proto//github.com/tetrafolium/luci-go/common/proto/config/project_config.proto", _config_pb = "config")
load("@proto//github.com/tetrafolium/luci-go/common/proto/realms/realms_config.proto", _realms_pb = "auth_service")
load("@proto//github.com/tetrafolium/luci-go/cv/api/config/v2/cq.proto", _cq_pb = "cq.config")
load("@proto//github.com/tetrafolium/luci-go/logdog/api/config/svcconfig/project.proto", _logdog_pb = "svcconfig")
load("@proto//github.com/tetrafolium/luci-go/luci_notify/api/config/notify.proto", _notify_pb = "notify")
load("@proto//github.com/tetrafolium/luci-go/milo/api/config/project.proto", _milo_pb = "milo")
load("@proto//github.com/tetrafolium/luci-go/resultdb/proto/v1/invocation.proto", _resultdb_pb = "luci.resultdb.v1")
load("@proto//github.com/tetrafolium/luci-go/resultdb/proto/v1/predicate.proto", _predicate_pb = "luci.resultdb.v1")
load("@proto//github.com/tetrafolium/luci-go/scheduler/appengine/messages/config.proto", _scheduler_pb = "scheduler.config")

buildbucket_pb = _buildbucket_pb
common_pb = _common_pb
config_pb = _config_pb
cq_pb = _cq_pb
logdog_pb = _logdog_pb
milo_pb = _milo_pb
notify_pb = _notify_pb
predicate_pb = _predicate_pb
realms_pb = _realms_pb
resultdb_pb = _resultdb_pb
scheduler_pb = _scheduler_pb
