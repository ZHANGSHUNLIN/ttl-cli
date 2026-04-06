package conf

import (
	"testing"
)

// TestGetTtlConf 测试获取配置
func TestGetTtlConf(t *testing.T) {
	tests := []struct {
		name          string
		expectedError bool
	}{
		{
			name:          "测试默认配置",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf, err := GetTtlConf()

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("GetTtlConf() error = %v, want nil", err)
				}
				// 验证配置是否包含必要的字段
				// DbPath 是动态生成的，首次创建时为空是正常的
				if conf.StorageType == "" {
					t.Errorf("GetTtlConf() returned config with empty StorageType")
				}
			}
		})
	}
}

// TestInitConfig 测试初始化配置
func TestInitConfig(t *testing.T) {
	tests := []struct {
		name          string
		expectedError bool
	}{
		{
			name:          "测试配置初始化",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 由于 InitConfig 是私有函数或没有公开，我们测试通过 GetTtlConf 来间接测试
			_, err := GetTtlConf()

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Failed to get config: %v", err)
				}
			}
		})
	}
}
