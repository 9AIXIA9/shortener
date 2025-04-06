package base62

import (
	"fmt"
	"math"
	"testing"
)

func TestConvert(t *testing.T) {
	tests := []struct {
		name   string
		number uint64
		want   string
	}{
		// 边界情况测试
		{name: "零值测试", number: 0, want: "0"},
		{name: "值为1", number: 1, want: "1"},
		{name: "值为9", number: 9, want: "9"},
		{name: "值为10", number: 10, want: "a"},
		{name: "值为35", number: 35, want: "z"},
		{name: "值为36", number: 36, want: "A"},
		{name: "值为61", number: 61, want: "Z"},

		// 现有测试用例
		{name: "值为63", number: 63, want: "11"},
		{name: "值为1163", number: 1163, want: "iL"},

		// 范围测试 - 修正这些值
		{name: "值为62", number: 62, want: "10"},
		{name: "值为3843", number: 3843, want: "ZZ"},  // 修正: 101 -> ZZ
		{name: "值为3844", number: 3844, want: "100"}, // 修正: 102 -> 100

		// 大值测试
		{name: "较大值测试", number: 9999999, want: "FXsj"},                // 修正
		{name: "更大值测试", number: 123456789, want: "8m0Kx"},             // 已正确
		{name: "接近uint32最大值", number: math.MaxUint32, want: "4GFfc3"}, // 修正
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Convert(tt.number)
			if got != tt.want {
				t.Errorf("Convert(%d) = %v, 期望结果 %v", tt.number, got, tt.want)
			}
		})
	}
}

// 手动计算结果的辅助测试
func TestManuallyCaculated(t *testing.T) {
	// 手动计算一些值 - 修正这些值
	numberToExpected := map[uint64]string{
		62*62 + 3:    "103",  // 已正确 (3847)
		62*62*62 + 1: "1001", // 已正确 (238329)
		100000:       "q0U",  // 修正: Q0u -> q0U
	}

	for number, expected := range numberToExpected {
		t.Run(fmt.Sprintf("值为%d", number), func(t *testing.T) {
			got := Convert(number)
			if got != expected {
				t.Errorf("Convert(%d) = %v, 期望结果 %v", number, got, expected)
			}
		})
	}
}

// 添加Parse函数的测试
func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
		want    uint64
		wantErr bool
	}{
		// 边界情况测试
		{name: "空字符串", encoded: "", want: 0, wantErr: true},
		{name: "值为0", encoded: "0", want: 0, wantErr: false},
		{name: "值为1", encoded: "1", want: 1, wantErr: false},
		{name: "值为a", encoded: "a", want: 10, wantErr: false},
		{name: "值为Z", encoded: "Z", want: 61, wantErr: false},

		// 多位��值测试
		{name: "值为10", encoded: "10", want: 62, wantErr: false},
		{name: "值为11", encoded: "11", want: 63, wantErr: false},
		{name: "值为101", encoded: "101", want: 3845, wantErr: false}, // 修正: 3843 -> 3845
		{name: "值为iL", encoded: "iL", want: 1163, wantErr: false},
		{name: "值为Q0u", encoded: "Q0u", want: 199918, wantErr: false}, // 修正: 100000 -> 199918

		// 错误输入测试
		{name: "包含无效字符", encoded: "abc@123", want: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%s) error = %v, wantErr %v", tt.encoded, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Parse(%s) = %v, want %v", tt.encoded, got, tt.want)
			}
		})
	}
}

// TestRoundTrip 测试编解码一致性
func TestRoundTrip(t *testing.T) {
	numbers := []uint64{0, 1, 10, 62, 63, 100, 1000, 3843, 3844, 9999, 123456789}

	for _, number := range numbers {
		t.Run(fmt.Sprintf("编解码一致性测试-%d", number), func(t *testing.T) {
			encoded := Convert(number)
			decoded, err := Parse(encoded)
			if err != nil {
				t.Errorf("Parse失败: %v", err)
			}
			if decoded != number {
				t.Errorf("编解码不一致: %d -> %s -> %d", number, encoded, decoded)
			}
		})
	}
}

// 性能测试
func BenchmarkConvert(b *testing.B) {
	benchmarks := []struct {
		name   string
		number uint64
	}{
		{"小值转换", 123},
		{"中值转换", 123456},
		{"大值转换", math.MaxUint32},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				Convert(bm.number)
			}
		})
	}
}

func BenchmarkParse(b *testing.B) {
	benchmarks := []struct {
		name    string
		encoded string
	}{
		{"小值解析", "7b"},     // 123
		{"中值解析", "w7E"},    // 123456
		{"大值解析", "4GFfc3"}, // MaxUint32
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = Parse(bm.encoded)
			}
		})
	}
}
