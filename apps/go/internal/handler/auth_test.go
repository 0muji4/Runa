package handler

import (
	"net/http"
	"strings"
	"testing"
)

func TestAuth_Signup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, h *Auth)
		body  string

		wantStatus  int
		wantCode    ErrorCode
		wantDetails int

		// check と wantCode/wantDetails は排他: どちらもボディを消費するため片方だけを設定する。
		check func(t *testing.T, res *http.Response)
	}{
		{
			name:        "有効なメールとパスワードでアカウントを作成する",
			setup:       nil,
			body:        `{"email":"a@b.com","password":"password123","display_name":"Runa"}`,
			wantStatus:  http.StatusCreated,
			wantCode:    "",
			wantDetails: -1,
			check: func(t *testing.T, res *http.Response) {
				checkTokensResponse(t, res, "a@b.com")
			},
		},
		{
			name:        "不正なメールと短いパスワードは検証エラー",
			setup:       nil,
			body:        `{"email":"not-an-email","password":"short"}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantDetails: 2,
			check:       nil,
		},
		{
			name: "重複メールは409",
			setup: func(t *testing.T, h *Auth) {
				res := postJSON(t, h.Signup, `{"email":"dup@b.com","password":"password123"}`)
				res.Body.Close()
			},
			body:        `{"email":"dup@b.com","password":"password123"}`,
			wantStatus:  http.StatusConflict,
			wantCode:    CodeEmailTaken,
			wantDetails: -1,
			check:       nil,
		},
		{
			name:        "未知のフィールドは拒否する",
			setup:       nil,
			body:        `{"email":"a@b.com","password":"password123","role":"admin"}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantDetails: -1,
			check:       nil,
		},
		{
			name:        "空白のみの表示名はメールのローカル部から補完する",
			setup:       nil,
			body:        `{"email":"trim@b.com","password":"password123","display_name":"` + strings.Repeat(" ", 3) + `"}`,
			wantStatus:  http.StatusCreated,
			wantCode:    "",
			wantDetails: -1,
			check: func(t *testing.T, res *http.Response) {
				got := decodeJSON[authTokensResponse](t, res)
				if got.User == nil {
					t.Fatal("user = nil, want the created user")
				}
				if got.User.DisplayName != "trim" {
					t.Errorf("user.display_name = %q, want %q", got.User.DisplayName, "trim")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newAuthHandler()
			if tt.setup != nil {
				tt.setup(t, h)
			}

			res := postJSON(t, h.Signup, tt.body)
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("POST signup %s = %d, want %d", tt.body, res.StatusCode, tt.wantStatus)
			}
			if got, want := res.Header.Get("Content-Type"), "application/json"; got != want {
				t.Errorf("POST signup Content-Type = %q, want %q", got, want)
			}
			checkErrorEnvelope(t, res, tt.wantCode, tt.wantDetails)
			if tt.check != nil {
				tt.check(t, res)
			}
		})
	}
}

func TestAuth_Login(t *testing.T) {
	t.Parallel()

	const email = "c@b.com"

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   ErrorCode
		check      func(t *testing.T, res *http.Response)
	}{
		{
			name:       "正しい資格情報でトークンを返す",
			body:       `{"email":"c@b.com","password":"password123"}`,
			wantStatus: http.StatusOK,
			wantCode:   "",
			check: func(t *testing.T, res *http.Response) {
				checkTokensResponse(t, res, email)
			},
		},
		{
			name:       "誤ったパスワードは拒否する",
			body:       `{"email":"c@b.com","password":"wrongpass"}`,
			wantStatus: http.StatusUnauthorized,
			wantCode:   CodeInvalidCredentials,
			check:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newAuthHandler()
			signup := postJSON(t, h.Signup, `{"email":"c@b.com","password":"password123"}`)
			signup.Body.Close()

			res := postJSON(t, h.Login, tt.body)
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("POST login %s = %d, want %d", tt.body, res.StatusCode, tt.wantStatus)
			}
			checkErrorEnvelope(t, res, tt.wantCode, -1)
			if tt.check != nil {
				tt.check(t, res)
			}
		})
	}
}
