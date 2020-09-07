# Copyright 2018 The LUCI Authors.
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

l = proto.new_loader(proto.new_descriptor_set(blob=read('./testprotos/all.pb')))
testprotos = l.module('github.com/tetrafolium/luci-go/starlark/starlarkproto/testprotos/test.proto')

TWO_POW_63 = 9223372036854775808
TWO_POW_31 = 2147483648

m1 = testprotos.SimpleFields()

# int64 max.
m1.i64 = TWO_POW_63 - 1  # still ok
assert.eq(m1.i64, TWO_POW_63 - 1)
assert.eq(proto.to_textpb(m1), "i64: 9223372036854775807\n")
def set_i64_large():
  m1.i64 = TWO_POW_63
assert.fails(set_i64_large, 'doesn\'t fit into int64')

# int64 min.
m1.i64 = -TWO_POW_63  # still ok
assert.eq(m1.i64, -TWO_POW_63)
assert.eq(proto.to_textpb(m1), "i64: -9223372036854775808\n")
def set_i64_small():
  m1.i64 = -TWO_POW_63-1
assert.fails(set_i64_small, 'doesn\'t fit into int64')

m2 = testprotos.SimpleFields()

# int32 max.
m2.i32 = TWO_POW_31 - 1  # still ok
assert.eq(m2.i32, TWO_POW_31 - 1)
assert.eq(proto.to_textpb(m2), "i32: 2147483647\n")
def set_i32_large():
  m2.i32 = TWO_POW_31
assert.fails(set_i32_large, 'doesn\'t fit into int32')

# int32 min.
m2.i32 = -TWO_POW_31  # still ok
assert.eq(m2.i32, -TWO_POW_31)
assert.eq(proto.to_textpb(m2), "i32: -2147483648\n")
def set_i32_small():
  m2.i32 = -TWO_POW_31-1
assert.fails(set_i32_small, 'doesn\'t fit into int32')

m3 = testprotos.SimpleFields()

# uint64 max.
m3.ui64 = 2*TWO_POW_63 - 1  # still ok
assert.eq(m3.ui64, 2*TWO_POW_63 - 1)
assert.eq(proto.to_textpb(m3), "ui64: 18446744073709551615\n")
def set_ui64_large():
  m3.ui64 = 2*TWO_POW_63
assert.fails(set_ui64_large, 'doesn\'t fit into uint64')

# uint64 min.
def set_ui64_small():
  m3.ui64 = -1
assert.fails(set_ui64_small, 'doesn\'t fit into uint64')

m4 = testprotos.SimpleFields()

# uint32 max.
m4.ui32 = 2*TWO_POW_31 - 1  # still ok
assert.eq(m4.ui32, 2*TWO_POW_31 - 1)
assert.eq(proto.to_textpb(m4), "ui32: 4294967295\n")
def set_ui32_large():
  m4.ui32 = 2*TWO_POW_31
assert.fails(set_ui32_large, 'doesn\'t fit into uint32')

# uint32 min.
def set_ui32_small():
  m4.ui32 = -1
assert.fails(set_ui32_small, 'doesn\'t fit into uint32')
