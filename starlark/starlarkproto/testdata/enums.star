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

# Enum constants follow C++ namespacing rules: they directly live in a namespace
# that defines the enum type itself. This also matches how proto enum are
# exposed in Python code.
#
# Values are represented by untyped integers. There's no type checks whatsoever
# when getting or setting enum-valued fields.

# Package-level enums.
assert.eq(testprotos.ENUM_DEFAULT, 0)
assert.eq(testprotos.ENUM_VAL_1, 1)

# Nested enums.
assert.eq(testprotos.Complex.UNKNOWN, 0)
assert.eq(testprotos.Complex.ENUM_VAL_1, 1)

m = testprotos.Complex()

# Enum valued field has a default.
assert.eq(m.enum_val, 0)

# Enum valued field can be set and read.
m.enum_val = testprotos.Complex.ENUM_VAL_1
assert.eq(m.enum_val, testprotos.Complex.ENUM_VAL_1)

# Can be reset.
m.enum_val = None
assert.eq(m.enum_val, 0)

# Per proto3 spec, enum-valued field can be set to an arbitrary int32 integer.
m.enum_val = 123
assert.eq(m.enum_val, 123)

# Serialization works.
assert.eq(
    proto.to_textpb(testprotos.Complex(enum_val=testprotos.Complex.ENUM_VAL_1)),
    "enum_val: ENUM_VAL_1\n")

# Setting to a wrong type fails.
def set_bad_val():
  m.enum_val = ''
assert.fails(set_bad_val, 'got string, want int')

# Attempting to overwrite enum constant fails.
def overwrite_global():
  testprotos.ENUM_DEFAULT = 10
assert.fails(overwrite_global, 'can\'t assign to .ENUM_DEFAULT field of module')
def overwrite_nested():
  testprotos.Complex.UNKNOWN = 10
assert.fails(overwrite_nested, 'can\'t assign to .UNKNOWN field of proto.MessageType')
