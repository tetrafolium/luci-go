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

l = proto.new_loader(proto.new_descriptor_set(blob=read('./testprotos/all.pb')))
testprotos = l.module('github.com/tetrafolium/luci-go/starlark/starlarkproto/testprotos/test.proto')

# Simple fields and slices of simple fields.
m = proto.from_textpb(testprotos.SimpleFields, """
i64: -64
i64_rep: 1
i64_rep: 2
i32: -32
ui64: 64
ui32: 32
b: true
f32: 2.0
f64: 3.0
s: "hello"
bs: "bytes"
bs_rep: "b0"
bs_rep: "b1"
""")
assert.eq(m.i64, -64)
assert.eq(list(m.i64_rep), [1, 2])
assert.eq(m.i32, -32)
assert.eq(m.ui64, 64)
assert.eq(m.ui32, 32)
assert.eq(m.b, True)
assert.eq(m.f32, 2.0)
assert.eq(m.f64, 3.0)
assert.eq(m.s, "hello")
assert.eq(m.bs, "bytes")
assert.eq(list(m.bs_rep), ["b0", "b1"])

# Enums.
m2 = proto.from_textpb(testprotos.Complex, "enum_val: ENUM_VAL_1")
assert.eq(m2.enum_val, testprotos.Complex.ENUM_VAL_1)

# Nested messages (singular and repeated).
m3 = proto.from_textpb(testprotos.MessageFields, """
single: <i: 123>
rep: <i: 456>
rep: <i: 789>
""")
assert.eq(m3.single.i, 123)
assert.eq(len(m3.rep), 2)
assert.eq(m3.rep[0].i, 456)
assert.eq(m3.rep[1].i, 789)

# Oneofs.
m4 = proto.from_textpb(testprotos.Complex, "simple: <i: 123>")
assert.eq(m4.simple.i, 123)
assert.eq(m4.another_simple, None)

# Maps with primitive values.
m5 = proto.from_textpb(testprotos.MapWithPrimitiveType, """
m1 <
  key: "abc"
  value: 1
>
m1 <
  key: "def"
  value: 2
>
""")
assert.eq(dict(m5.m1), {'abc': 1, 'def': 2})

# Maps with message values.
m6 = proto.from_textpb(testprotos.MapWithMessageType, """
m <
  key: "abc"
  value: <i: 1>
>
m <
  key: "def"
  value: <i: 2>
>
""")
assert.eq(len(m6.m), 2)
assert.eq(m6.m['abc'].i, 1)
assert.eq(m6.m['def'].i, 2)
