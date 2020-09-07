// Copyright 2018 The LUCI Authors.
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

package buildbucketpb

//go:generate go install github.com/tetrafolium/luci-go/grpc/cmd/cproto github.com/tetrafolium/luci-go/grpc/cmd/svcdec
//go:generate cproto
//go:generate proto-gae -type Bucket -type Builder -type Build -type BuildInfra
//go:generate mockgen -source builds_service.pb.go -destination builds_service.mock.pb.go -package buildbucketpb -write_package_comment=false
//go:generate goimports -w builds_service.mock.pb.go
//go:generate svcdec -type BuildsServer -type BuildersServer
