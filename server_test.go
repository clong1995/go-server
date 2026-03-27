package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	// 注册一个测试用的 handle
	testHandle := Handle{
		Uri:  "/test",
		Desc: "test handle",
		Process: func(uid int64, param Param) (any, error) {
			return "ok", nil
		},
	}
	testHandle.Register()

	tests := []struct {
		name           string
		path           string
		userId         string
		wantStatusCode int
		wantBody       string
	}{
		{
			name:           "handler not found",
			path:           "/not-found",
			userId:         "123",
			wantStatusCode: http.StatusNotFound,
			wantBody:       "handler not found\n",
		},
		{
			name:           "user id is empty",
			path:           "/test",
			userId:         "",
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "user id is empty\n",
		},
		{
			name:           "invalid user id",
			path:           "/test",
			userId:         "abc",
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "invalid user id\n",
		},
		{
			name:           "success",
			path:           "/test",
			userId:         "123",
			wantStatusCode: http.StatusOK,
			// 注意：由于响应是 JSON 格式，并且时间戳是动态的，我们只检查静态部分
			wantBody: `{"State":"OK","Data":"ok"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tt.userId != "" {
				req.Header.Set("user-id", tt.userId)
			}

			rr := httptest.NewRecorder()
			http.HandlerFunc(handler).ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.wantStatusCode)
			}

			// 对于成功的 case，我们只检查 body 的前缀
			if tt.name == "success" {
				if !strings.HasPrefix(rr.Body.String(), tt.wantBody) {
					t.Errorf("handler returned unexpected body prefix: got %v want %v",
						rr.Body.String(), tt.wantBody)
				}
			} else {
				if rr.Body.String() != tt.wantBody {
					t.Errorf("handler returned unexpected body: got %v want %v",
						rr.Body.String(), tt.wantBody)
				}
			}
		})
	}
}
