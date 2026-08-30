package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/service"
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
				if _, err := svc.SignupEmail(ctx, tt.seedEmail, "password123", ""); err != nil {
					t.Fatalf("seeding %q: SignupEmail() error = %v, want nil", tt.seedEmail, err)
				}
			}

			res, err := svc.SignupEmail(ctx, tt.email, tt.password, tt.displayName)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("SignupEmail(%q) error = %v, want %v", tt.email, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("SignupEmail(%q) error = %v, want nil", tt.email, err)
			}
			if res.Tokens.AccessToken == "" {
				t.Error("SignupEmail() access token is empty, want a token")
			}
			if res.Tokens.RefreshToken == "" {
				t.Error("SignupEmail() refresh token is empty, want a token")
			}
			if res.User.Email == nil {
				t.Fatalf("SignupEmail(%q) user.email = nil, want %q", tt.email, tt.wantEmail)
			}
			if *res.User.Email != tt.wantEmail {
				t.Errorf("SignupEmail(%q) user.email = %q, want %q",
					tt.email, *res.User.Email, tt.wantEmail)
			}
			if res.User.AuthProvider != tt.wantProvider {
				t.Errorf("SignupEmail(%q) auth_provider = %q, want %q",
					tt.email, res.User.AuthProvider, tt.wantProvider)
			}
			if res.User.DisplayName != tt.wantDisplayName {
				t.Errorf("SignupEmail(%q, display_name=%q) display_name = %q, want %q",
					tt.email, tt.displayName, res.User.DisplayName, tt.wantDisplayName)
			}
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
			if _, err := svc.SignupEmail(ctx, "user@example.com", "password123", ""); err != nil {
				t.Fatalf("SignupEmail() error = %v, want nil", err)
			}

			res, err := svc.LoginEmail(ctx, tt.email, tt.password)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("LoginEmail(%q) error = %v, want %v", tt.email, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoginEmail(%q) error = %v, want nil", tt.email, err)
			}
			if res.Tokens.AccessToken == "" {
				t.Error("LoginEmail() access token is empty, want a token")
			}
			if res.Tokens.RefreshToken == "" {
				t.Error("LoginEmail() refresh token is empty, want a token")
			}
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
		if err != nil {
			t.Fatalf("SignupEmail() error = %v, want nil", err)
		}
		first = res.Tokens.RefreshToken
		next, err := svc.Refresh(ctx, first)
		if err != nil {
			t.Fatalf("Refresh() error = %v, want nil", err)
		}
		if next.RefreshToken == first {
			t.Fatalf("Refresh() returned the same refresh token %q; it must rotate", first)
		}
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

			token := tt.token(first, rotated)
			_, err := svc.Refresh(ctx, token)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Refresh(%q) error = %v, want %v", token, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Refresh(%q) error = %v, want nil", token, err)
			}
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
			if err != nil {
				t.Fatalf("SignupEmail() error = %v, want nil", err)
			}
			token := res.Tokens.RefreshToken

			// Logout is idempotent: repeating it must stay a no-op, not an error.
			for i := 0; i < tt.logoutCount; i++ {
				if err := svc.Logout(ctx, token); err != nil {
					t.Fatalf("Logout() call %d error = %v, want nil", i+1, err)
				}
			}

			if _, err := svc.Refresh(ctx, token); !errors.Is(err, tt.wantRefreshErr) {
				t.Errorf("Refresh() after %d logout(s) error = %v, want %v",
					tt.logoutCount, err, tt.wantRefreshErr)
			}
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
			if err != nil {
				t.Fatalf("SignupEmail() error = %v, want nil", err)
			}

			userID := res.User.ID
			if !tt.useRealID {
				userID = "does-not-exist"
			}

			got, err := svc.Me(ctx, userID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Me(%q) error = %v, want %v", userID, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Me(%q) error = %v, want nil", userID, err)
			}
			if got.ID != userID {
				t.Errorf("Me(%q) id = %q, want %q", userID, got.ID, userID)
			}
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
				if err != nil {
					t.Fatalf("LoginApple() error = %v, want nil", err)
				}
				if res.User.AuthProvider != "apple" {
					t.Errorf("LoginApple() auth_provider = %q, want %q", res.User.AuthProvider, "apple")
				}
				if res.User.AppleSub == nil {
					t.Error("LoginApple() user.apple_sub = nil, want the provider subject")
				}
			},
		},
		{
			name: "再サインインは同じユーザーを再利用する",
			run: func(t *testing.T) {
				svc := newAppleService()
				ctx := context.Background()

				first, err := svc.LoginApple(ctx, "id-token", "")
				if err != nil {
					t.Fatalf("first LoginApple() error = %v, want nil", err)
				}
				second, err := svc.LoginApple(ctx, "id-token", "")
				if err != nil {
					t.Fatalf("second LoginApple() error = %v, want nil", err)
				}
				if second.User.ID != first.User.ID {
					t.Errorf("second LoginApple() user id = %q, want the first login's id %q",
						second.User.ID, first.User.ID)
				}
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
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("LoginGoogle() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoginGoogle() error = %v, want nil", err)
			}
			if res.User.AuthProvider != "google" {
				t.Errorf("LoginGoogle() auth_provider = %q, want %q", res.User.AuthProvider, "google")
			}
			if res.User.GoogleSub == nil {
				t.Error("LoginGoogle() user.google_sub = nil, want the provider subject")
			}
		})
	}
}
