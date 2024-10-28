package server

import "testing"

func TestListenAndServe(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			"启动服务",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Listen()
		})
	}
}
