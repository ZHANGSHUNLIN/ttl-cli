package conf

import (
	"testing"
)

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
