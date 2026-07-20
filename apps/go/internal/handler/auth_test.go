package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				got := decodeJSON[authTokensResponse](t, res)
				assert.NotEmpty(t, got.AccessToken)
				assert.NotEmpty(t, got.RefreshToken)
				assert.Equal(t, "Bearer", got.TokenType)
				require.NotNil(t, got.User)
				require.NotNil(t, got.User.Email)
				assert.Equal(t, "a@b.com", *got.User.Email)
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
				require.NotNil(t, got.User)
				assert.Equal(t, "trim", got.User.DisplayName)
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

			require.Equal(t, tt.wantStatus, res.StatusCode)
			assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
			if tt.wantCode != "" || tt.wantDetails >= 0 {
				env := decodeError(t, res)
				if tt.wantCode != "" {
					assert.Equal(t, tt.wantCode, env.Error.Code)
				}
				if tt.wantDetails >= 0 {
					assert.Len(t, env.Error.Details, tt.wantDetails)
				}
			}
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
				got := decodeJSON[authTokensResponse](t, res)
				assert.NotEmpty(t, got.AccessToken)
				assert.NotEmpty(t, got.RefreshToken)
				assert.Equal(t, "Bearer", got.TokenType)
				require.NotNil(t, got.User)
				require.NotNil(t, got.User.Email)
				assert.Equal(t, email, *got.User.Email)
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

			require.Equal(t, tt.wantStatus, res.StatusCode)
			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, decodeError(t, res).Error.Code)
			}
			if tt.check != nil {
				tt.check(t, res)
			}
		})
	}
}
