package service_test

import (
	"context"
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubVerifier struct {
	id  auth.OIDCIdentity
	err error
}

func (s stubVerifier) Verify(context.Context, string) (auth.OIDCIdentity, error) {
	return s.id, s.err
}

func TestAuthService_SignupEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		seedEmail       string
		email           string
		password        string
		displayName     string
		wantErr         error
		wantEmail       string
		wantProvider    string
		wantDisplayName string
	}{
		{
			name:            "メールを正規化しトークンを返す",
			seedEmail:       "",
			email:           "User@Example.com",
			password:        "password123",
			displayName:     "Runa",
			wantErr:         nil,
			wantEmail:       "user@example.com",
			wantProvider:    "email",
			wantDisplayName: "Runa",
		},
		{
			name:            "表示名が空ならメールのローカル部から補う",
			seedEmail:       "",
			email:           "moon@example.com",
			password:        "password123",
			displayName:     "",
			wantErr:         nil,
			wantEmail:       "moon@example.com",
			wantProvider:    "email",
			wantDisplayName: "moon",
		},
		{
			name:            "重複メールはErrEmailTaken",
			seedEmail:       "dup@example.com",
			email:           "dup@example.com",
			password:        "password123",
			displayName:     "",
			wantErr:         service.ErrEmailTaken,
			wantEmail:       "",
			wantProvider:    "",
			wantDisplayName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newAuthService(memauth.New(), nil, nil)
			ctx := context.Background()

			if tt.seedEmail != "" {
				_, err := svc.SignupEmail(ctx, tt.seedEmail, "password123", "")
				require.NoError(t, err)
			}

			res, err := svc.SignupEmail(ctx, tt.email, tt.password, tt.displayName)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, res.Tokens.AccessToken)
			assert.NotEmpty(t, res.Tokens.RefreshToken)
			require.NotNil(t, res.User.Email)
			assert.Equal(t, tt.wantEmail, *res.User.Email)
			assert.Equal(t, tt.wantProvider, res.User.AuthProvider)
			assert.Equal(t, tt.wantDisplayName, res.User.DisplayName)
		})
	}
}

func TestAuthService_LoginEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{
			name:     "正しい資格情報でログインできる",
			email:    "user@example.com",
			password: "password123",
			wantErr:  nil,
		},
		{
			name:     "誤ったパスワードはErrInvalidCredentials",
			email:    "user@example.com",
			password: "wrong",
			wantErr:  service.ErrInvalidCredentials,
		},
		{
			name:     "未登録ユーザーはErrInvalidCredentials",
			email:    "missing@example.com",
			password: "whatever",
			wantErr:  service.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newAuthService(memauth.New(), nil, nil)
			ctx := context.Background()
			_, err := svc.SignupEmail(ctx, "user@example.com", "password123", "")
			require.NoError(t, err)

			res, err := svc.LoginEmail(ctx, tt.email, tt.password)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, res.Tokens.AccessToken)
			assert.NotEmpty(t, res.Tokens.RefreshToken)
		})
	}
}

func TestAuthService_Refresh(t *testing.T) {
	t.Parallel()

	// setup signs up, rotates once and returns (first token, rotated token); the
	// rotation itself is asserted here so every case inherits it.
	setup := func(t *testing.T) (svc *service.AuthService, ctx context.Context, first, rotated string) {
		t.Helper()
		svc = newAuthService(memauth.New(), nil, nil)
		ctx = context.Background()
		res, err := svc.SignupEmail(ctx, "rot@example.com", "password123", "")
		require.NoError(t, err)
		first = res.Tokens.RefreshToken
		next, err := svc.Refresh(ctx, first)
		require.NoError(t, err)
		require.NotEqual(t, first, next.RefreshToken)
		return svc, ctx, first, next.RefreshToken
	}

	tests := []struct {
		name    string
		token   func(first, rotated string) string
		wantErr error
	}{
		{
			name:    "ローテーション後のトークンは有効",
			token:   func(_, rotated string) string { return rotated },
			wantErr: nil,
		},
		{
			name:    "使用済みの旧トークンはErrInvalidRefreshToken",
			token:   func(first, _ string) string { return first },
			wantErr: service.ErrInvalidRefreshToken,
		},
		{
			name:    "未知のトークンはErrInvalidRefreshToken",
			token:   func(_, _ string) string { return "unknown-token" },
			wantErr: service.ErrInvalidRefreshToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, ctx, first, rotated := setup(t)

			_, err := svc.Refresh(ctx, tt.token(first, rotated))
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		logoutCount    int
		wantRefreshErr error
	}{
		{
			name:           "リフレッシュトークンを失効させる",
			logoutCount:    1,
			wantRefreshErr: service.ErrInvalidRefreshToken,
		},
		{
			name:           "二重ログアウトは冪等",
			logoutCount:    2,
			wantRefreshErr: service.ErrInvalidRefreshToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newAuthService(memauth.New(), nil, nil)
			ctx := context.Background()
			res, err := svc.SignupEmail(ctx, "out@example.com", "password123", "")
			require.NoError(t, err)
			token := res.Tokens.RefreshToken

			for i := 0; i < tt.logoutCount; i++ {
				assert.NoError(t, svc.Logout(ctx, token))
			}

			_, err = svc.Refresh(ctx, token)
			assert.ErrorIs(t, err, tt.wantRefreshErr)
		})
	}
}

func TestAuthService_Me(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		useRealID bool
		wantErr   error
	}{
		{
			name:      "認証済みユーザーを返す",
			useRealID: true,
			wantErr:   nil,
		},
		{
			name:      "存在しないユーザーはErrUserNotFound",
			useRealID: false,
			wantErr:   service.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newAuthService(memauth.New(), nil, nil)
			ctx := context.Background()
			res, err := svc.SignupEmail(ctx, "me@example.com", "password123", "")
			require.NoError(t, err)

			userID := res.User.ID
			if !tt.useRealID {
				userID = "does-not-exist"
			}

			got, err := svc.Me(ctx, userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, userID, got.ID)
		})
	}
}

func TestAuthService_LoginApple(t *testing.T) {
	t.Parallel()

	newAppleService := func() *service.AuthService {
		apple := stubVerifier{id: auth.OIDCIdentity{Subject: "apple-sub-9", Email: "a@example.com", Name: "Apple User"}}
		return newAuthService(memauth.New(), apple, nil)
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "初回サインインでappleユーザーを作成する",
			run: func(t *testing.T) {
				svc := newAppleService()

				res, err := svc.LoginApple(context.Background(), "id-token", "")
				require.NoError(t, err)
				assert.Equal(t, "apple", res.User.AuthProvider)
				assert.NotNil(t, res.User.AppleSub)
			},
		},
		{
			name: "再サインインは同じユーザーを再利用する",
			run: func(t *testing.T) {
				svc := newAppleService()
				ctx := context.Background()

				first, err := svc.LoginApple(ctx, "id-token", "")
				require.NoError(t, err)
				second, err := svc.LoginApple(ctx, "id-token", "")
				require.NoError(t, err)
				assert.Equal(t, first.User.ID, second.User.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestAuthService_LoginGoogle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		verifier stubVerifier
		wantErr  error
	}{
		{
			name:     "検証エラーはそのまま伝播する",
			verifier: stubVerifier{err: auth.ErrProviderVerification},
			wantErr:  auth.ErrProviderVerification,
		},
		{
			name:     "有効なトークンでサインインできる",
			verifier: stubVerifier{id: auth.OIDCIdentity{Subject: "g-sub-1", Email: "g@example.com"}},
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newAuthService(memauth.New(), nil, tt.verifier)

			res, err := svc.LoginGoogle(context.Background(), "token")
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "google", res.User.AuthProvider)
			assert.NotNil(t, res.User.GoogleSub)
		})
	}
}
