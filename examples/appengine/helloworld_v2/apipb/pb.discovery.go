// Code generated by cproto. DO NOT EDIT.

package apipb

import discovery "github.com/tetrafolium/luci-go/grpc/discovery"

import "github.com/golang/protobuf/protoc-gen-go/descriptor"

func init() {
	discovery.RegisterDescriptorSetCompressed(
		[]string{
			"luci.examples.helloworld.Greeter",
		},
		[]byte{31, 139,
			8, 0, 0, 0, 0, 0, 0, 255, 164, 86, 79, 111, 219, 200,
			21, 167, 37, 199, 118, 103, 221, 221, 64, 45, 226, 69, 90, 100,
			223, 186, 217, 38, 105, 21, 202, 112, 46, 109, 2, 20, 160, 196,
			177, 53, 89, 154, 84, 73, 202, 94, 239, 165, 161, 200, 145, 52,
			5, 201, 97, 103, 134, 118, 132, 197, 126, 153, 94, 10, 244, 19,
			180, 199, 2, 61, 244, 208, 239, 210, 107, 129, 2, 69, 49, 67,
			202, 150, 189, 72, 123, 168, 46, 26, 190, 55, 243, 254, 252, 230,
			205, 251, 61, 244, 167, 30, 250, 209, 130, 243, 69, 78, 7, 149,
			224, 138, 207, 234, 249, 128, 22, 149, 90, 217, 230, 179, 247, 73,
			163, 180, 215, 202, 195, 93, 244, 0, 107, 253, 240, 10, 253, 32,
			229, 133, 125, 79, 63, 68, 70, 59, 209, 159, 147, 173, 175, 159,
			45, 152, 90, 214, 51, 59, 229, 197, 96, 193, 243, 164, 92, 220,
			186, 169, 212, 170, 162, 178, 241, 246, 207, 173, 173, 223, 119, 186,
			167, 147, 225, 31, 59, 79, 78, 27, 139, 147, 118, 159, 125, 65,
			243, 252, 203, 146, 95, 151, 177, 222, 255, 246, 223, 15, 209, 78,
			111, 251, 137, 245, 234, 33, 250, 251, 62, 218, 218, 239, 117, 159,
			88, 189, 227, 191, 236, 131, 57, 144, 242, 28, 134, 245, 124, 78,
			133, 132, 151, 208, 152, 122, 38, 33, 75, 84, 2, 172, 84, 84,
			164, 203, 164, 92, 80, 152, 115, 81, 36, 10, 193, 136, 87, 43,
			193, 22, 75, 5, 199, 71, 71, 191, 104, 15, 0, 41, 83, 27,
			192, 201, 115, 48, 58, 9, 130, 74, 42, 174, 104, 102, 35, 88,
			42, 85, 201, 215, 131, 65, 70, 175, 104, 206, 43, 42, 228, 26,
			3, 157, 100, 213, 6, 241, 114, 214, 4, 49, 64, 8, 66, 154,
			49, 169, 4, 155, 213, 138, 241, 18, 146, 50, 131, 90, 82, 96,
			37, 72, 94, 139, 148, 26, 201, 140, 149, 137, 88, 153, 184, 100,
			31, 174, 153, 90, 2, 23, 230, 159, 215, 10, 65, 193, 51, 54,
			103, 105, 162, 45, 244, 33, 17, 20, 42, 42, 10, 166, 20, 205,
			160, 18, 252, 138, 101, 52, 3, 181, 76, 20, 168, 165, 206, 46,
			207, 249, 53, 43, 23, 144, 242, 50, 99, 250, 144, 212, 135, 16,
			20, 84, 189, 70, 8, 244, 239, 103, 247, 2, 147, 192, 231, 235,
			136, 82, 158, 81, 40, 106, 169, 64, 80, 149, 176, 210, 88, 77,
			102, 252, 74, 171, 90, 196, 16, 148, 92, 177, 148, 246, 65, 45,
			153, 132, 156, 73, 165, 45, 108, 122, 44, 179, 123, 225, 100, 76,
			166, 121, 194, 10, 42, 236, 15, 5, 193, 202, 77, 44, 214, 65,
			84, 130, 103, 117, 74, 111, 227, 64, 183, 129, 252, 95, 113, 32,
			104, 179, 203, 120, 90, 23, 180, 84, 201, 250, 146, 6, 92, 0,
			87, 75, 42, 160, 72, 20, 21, 44, 201, 229, 45, 212, 230, 130,
			212, 146, 34, 216, 140, 254, 38, 41, 159, 50, 115, 82, 27, 46,
			147, 130, 234, 128, 54, 107, 171, 228, 183, 58, 131, 59, 83, 82,
			103, 84, 54, 166, 184, 144, 80, 36, 43, 152, 81, 93, 41, 25,
			40, 14, 180, 204, 184, 144, 84, 23, 69, 37, 120, 193, 21, 133,
			6, 19, 37, 33, 163, 130, 93, 209, 12, 230, 130, 23, 168, 65,
			65, 242, 185, 186, 214, 101, 210, 86, 16, 200, 138, 166, 186, 130,
			160, 18, 76, 23, 150, 208, 181, 83, 54, 85, 36, 165, 137, 29,
			65, 60, 38, 17, 68, 193, 73, 124, 225, 132, 24, 72, 4, 147,
			48, 56, 39, 46, 118, 97, 120, 9, 241, 24, 195, 40, 152, 92,
			134, 228, 116, 28, 195, 56, 240, 92, 28, 70, 224, 248, 46, 140,
			2, 63, 14, 201, 112, 26, 7, 97, 132, 224, 208, 137, 128, 68,
			135, 70, 227, 248, 151, 128, 191, 154, 132, 56, 138, 32, 8, 129,
			156, 77, 60, 130, 93, 184, 112, 194, 208, 241, 99, 130, 163, 62,
			16, 127, 228, 77, 93, 226, 159, 246, 97, 56, 141, 193, 15, 98,
			4, 30, 57, 35, 49, 118, 33, 14, 250, 198, 237, 119, 207, 65,
			112, 2, 103, 56, 28, 141, 29, 63, 118, 134, 196, 35, 241, 165,
			113, 120, 66, 98, 95, 59, 59, 9, 66, 4, 14, 76, 156, 48,
			38, 163, 169, 231, 132, 48, 153, 134, 147, 32, 194, 160, 51, 115,
			73, 52, 242, 28, 114, 134, 93, 27, 136, 15, 126, 0, 248, 28,
			251, 49, 68, 99, 199, 243, 238, 38, 138, 32, 184, 240, 113, 168,
			163, 223, 76, 19, 134, 24, 60, 226, 12, 61, 172, 93, 153, 60,
			93, 18, 226, 81, 172, 19, 186, 93, 141, 136, 139, 253, 216, 241,
			250, 8, 162, 9, 30, 17, 199, 235, 3, 254, 10, 159, 77, 60,
			39, 188, 236, 183, 70, 35, 252, 235, 41, 246, 99, 226, 120, 224,
			58, 103, 206, 41, 142, 224, 249, 255, 66, 101, 18, 6, 163, 105,
			136, 207, 116, 212, 193, 9, 68, 211, 97, 20, 147, 120, 26, 99,
			56, 13, 2, 215, 128, 29, 225, 240, 156, 140, 112, 244, 6, 188,
			32, 50, 128, 77, 35, 220, 71, 224, 58, 177, 99, 92, 79, 194,
			224, 132, 196, 209, 27, 189, 30, 78, 35, 98, 128, 35, 126, 140,
			195, 112, 58, 137, 73, 224, 191, 128, 113, 112, 129, 207, 113, 8,
			35, 103, 26, 97, 215, 32, 28, 248, 58, 91, 93, 43, 56, 8,
			47, 181, 89, 141, 131, 185, 129, 62, 92, 140, 113, 60, 198, 161,
			6, 213, 160, 229, 104, 24, 162, 56, 36, 163, 120, 115, 91, 16,
			66, 28, 132, 49, 218, 200, 19, 124, 124, 234, 145, 83, 236, 143,
			176, 86, 7, 218, 204, 5, 137, 240, 11, 112, 66, 18, 233, 13,
			196, 56, 134, 11, 231, 18, 130, 169, 201, 90, 95, 212, 52, 194,
			168, 89, 111, 148, 110, 223, 220, 39, 144, 19, 112, 220, 115, 162,
			35, 111, 119, 79, 130, 40, 34, 109, 185, 24, 216, 70, 227, 22,
			115, 27, 161, 61, 180, 213, 233, 117, 97, 239, 64, 175, 246, 122,
			221, 67, 235, 13, 250, 30, 234, 236, 125, 209, 44, 27, 225, 79,
			172, 95, 25, 225, 71, 205, 178, 17, 62, 181, 250, 70, 184, 213,
			44, 27, 225, 23, 214, 207, 141, 176, 93, 54, 194, 159, 90, 135,
			70, 136, 154, 101, 35, 124, 102, 125, 110, 132, 79, 155, 101, 35,
			124, 110, 125, 102, 132, 159, 53, 203, 127, 117, 80, 103, 219, 234,
			117, 95, 89, 15, 31, 255, 163, 3, 14, 44, 104, 73, 5, 75,
			193, 240, 39, 20, 84, 202, 100, 65, 27, 10, 88, 241, 26, 210,
			164, 4, 65, 95, 106, 162, 81, 28, 146, 43, 206, 50, 200, 232,
			156, 149, 166, 253, 213, 85, 174, 201, 132, 102, 232, 238, 121, 211,
			126, 87, 188, 22, 224, 76, 136, 180, 193, 1, 181, 170, 88, 154,
			228, 64, 223, 39, 69, 149, 83, 96, 82, 219, 51, 252, 165, 32,
			145, 166, 139, 9, 250, 187, 154, 74, 133, 160, 237, 106, 130, 202,
			138, 151, 218, 243, 170, 50, 173, 47, 41, 181, 61, 77, 62, 75,
			158, 217, 112, 194, 5, 176, 82, 170, 164, 76, 233, 154, 141, 52,
			191, 178, 148, 194, 9, 231, 240, 77, 35, 2, 16, 85, 10, 195,
			68, 60, 191, 55, 100, 216, 102, 198, 120, 161, 185, 169, 22, 165,
			132, 15, 232, 223, 52, 102, 190, 213, 141, 109, 73, 225, 109, 20,
			248, 134, 73, 168, 188, 105, 243, 115, 46, 224, 157, 217, 253, 78,
			103, 214, 96, 97, 54, 242, 217, 111, 105, 170, 224, 221, 55, 223,
			190, 179, 17, 66, 168, 187, 109, 109, 245, 186, 175, 246, 190, 63,
			219, 49, 110, 94, 161, 63, 236, 34, 111, 193, 237, 116, 41, 120,
			193, 234, 194, 230, 98, 49, 200, 235, 148, 13, 90, 168, 228, 32,
			169, 42, 90, 46, 88, 73, 7, 75, 170, 153, 135, 139, 60, 251,
			205, 213, 241, 32, 169, 88, 53, 219, 144, 181, 179, 214, 167, 250,
			180, 189, 62, 109, 223, 234, 31, 255, 183, 17, 237, 216, 69, 187,
			167, 130, 82, 69, 69, 239, 151, 232, 65, 148, 172, 198, 172, 247,
			232, 254, 92, 214, 64, 242, 248, 3, 242, 67, 107, 184, 251, 245,
			3, 19, 214, 219, 191, 61, 208, 243, 213, 199, 214, 167, 91, 232,
			175, 219, 102, 190, 250, 216, 234, 29, 255, 121, 251, 206, 168, 116,
			124, 100, 32, 245, 166, 35, 2, 78, 173, 150, 92, 72, 205, 31,
			30, 75, 105, 169, 9, 171, 46, 179, 150, 253, 156, 42, 73, 245,
			206, 70, 211, 135, 115, 42, 52, 219, 192, 177, 125, 4, 207, 245,
			134, 195, 86, 117, 168, 239, 75, 87, 174, 38, 190, 146, 43, 83,
			99, 134, 203, 230, 44, 167, 64, 223, 167, 180, 82, 186, 60, 83,
			94, 84, 57, 211, 181, 115, 195, 194, 107, 243, 54, 130, 203, 214,
			2, 159, 153, 185, 37, 49, 99, 130, 174, 193, 141, 109, 144, 168,
			182, 234, 204, 52, 247, 122, 48, 184, 190, 190, 182, 19, 19, 105,
			115, 141, 205, 62, 57, 240, 200, 8, 251, 17, 126, 121, 108, 31,
			33, 4, 211, 50, 167, 82, 154, 114, 103, 130, 102, 48, 91, 65,
			82, 153, 151, 52, 203, 41, 228, 201, 181, 126, 0, 201, 66, 208,
			134, 178, 89, 105, 104, 150, 149, 139, 254, 13, 31, 111, 204, 11,
			119, 96, 90, 71, 198, 228, 157, 13, 102, 18, 185, 97, 212, 161,
			19, 145, 168, 143, 224, 130, 196, 99, 221, 2, 55, 233, 208, 48,
			137, 75, 116, 219, 54, 189, 94, 183, 202, 47, 137, 239, 246, 161,
			29, 69, 232, 123, 93, 249, 82, 135, 200, 52, 128, 102, 152, 141,
			40, 189, 227, 126, 222, 190, 224, 155, 105, 65, 143, 236, 181, 110,
			44, 11, 126, 69, 133, 105, 30, 183, 35, 131, 153, 172, 16, 228,
			172, 96, 205, 123, 146, 223, 205, 232, 166, 175, 62, 220, 131, 182,
			179, 245, 172, 31, 175, 91, 104, 187, 236, 90, 189, 238, 15, 119,
			159, 34, 132, 58, 59, 86, 111, 251, 145, 46, 62, 132, 186, 59,
			250, 201, 61, 218, 251, 4, 125, 132, 182, 119, 172, 142, 213, 235,
			30, 116, 48, 218, 71, 15, 244, 199, 86, 175, 123, 176, 243, 209,
			250, 171, 211, 235, 30, 236, 127, 190, 254, 234, 246, 186, 7, 125,
			103, 253, 82, 255, 19, 0, 0, 255, 255, 124, 127, 247, 67, 229,
			12, 0, 0},
	)
}

// FileDescriptorSet returns a descriptor set for this proto package, which
// includes all defined services, and all transitive dependencies.
//
// Will not return nil.
//
// Do NOT modify the returned descriptor.
func FileDescriptorSet() *descriptor.FileDescriptorSet {
	// We just need ONE of the service names to look up the FileDescriptorSet.
	ret, err := discovery.GetDescriptorSet("luci.examples.helloworld.Greeter")
	if err != nil {
		panic(err)
	}
	return ret
}
