package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/openmcpdirectory/omdr/internal/cli/client"
	"github.com/openmcpdirectory/omdr/internal/cli/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	callbackPort = 8765
	tokenKey     = "auth.token"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  "Authenticate with the OMDR registry to publish servers and access authenticated features",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the OMDR registry",
	Long:  "Open a browser to authenticate with GitHub OAuth and store the token locally",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := viper.GetString("api_url")
		if apiURL == "" {
			apiURL = "http://localhost:8080"
		}

		apiClient := client.NewClient(apiURL)

		// Start local callback server
		tokenChan := make(chan string, 1)
		errChan := make(chan error, 1)

		srv := startCallbackServer(tokenChan, errChan)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			srv.Shutdown(ctx)
		}()

		// Get auth URL from API
		var authResp struct {
			AuthURL string `json:"auth_url"`
			State   string `json:"state"`
		}

		if err := apiClient.Get("/api/v1/auth/cli", &authResp); err != nil {
			return fmt.Errorf("getting auth URL: %w", err)
		}

		// Open browser
		fmt.Println("Opening browser for authentication...")
		if err := openBrowser(authResp.AuthURL); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open browser automatically. Please visit:\n%s\n", authResp.AuthURL)
		}

		// Wait for callback or timeout
		select {
		case token := <-tokenChan:
			// Store token in config
			mgr, err := config.NewManager()
			if err != nil {
				return fmt.Errorf("initializing config manager: %w", err)
			}

			if err := mgr.Set(tokenKey, token); err != nil {
				return fmt.Errorf("storing token: %w", err)
			}

			// Get user info to display username
			apiClient.SetToken(token)
			var user struct {
				Username string `json:"username"`
			}
			if err := apiClient.Get("/api/v1/users/me", &user); err != nil {
				fmt.Println("Authentication successful!")
				return nil
			}

			fmt.Printf("Authentication successful! Logged in as %s\n", user.Username)
			return nil

		case err := <-errChan:
			return fmt.Errorf("authentication failed: %w", err)

		case <-time.After(5 * time.Minute):
			return fmt.Errorf("authentication timeout - no callback received")
		}
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored credentials",
	Long:  "Remove the authentication token from local configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("initializing config manager: %w", err)
		}

		// Set empty value to clear the token
		if err := mgr.Set(tokenKey, ""); err != nil {
			return fmt.Errorf("clearing token: %w", err)
		}

		fmt.Println("Logged out successfully")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display current authentication state",
	Long:  "Show whether you are authenticated and display your username if logged in",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("initializing config manager: %w", err)
		}

		token, err := mgr.Get(tokenKey)
		if err != nil || token == "" {
			fmt.Println("Not authenticated")
			fmt.Println("Run 'omdr auth login' to authenticate")
			return nil
		}

		// Parse token to check expiration
		parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
			return []byte("dummy"), nil // We don't validate signature here, just parse
		})

		if err != nil && parsedToken == nil {
			fmt.Println("Authentication token is invalid")
			fmt.Println("Run 'omdr auth login' to re-authenticate")
			return nil
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			fmt.Println("Authentication token is invalid")
			fmt.Println("Run 'omdr auth login' to re-authenticate")
			return nil
		}

		// Check expiration
		exp, ok := claims["exp"].(float64)
		if ok {
			expTime := time.Unix(int64(exp), 0)
			if time.Now().After(expTime) {
				fmt.Println("Authentication token has expired")
				fmt.Printf("Expired at: %s\n", expTime.Format(time.RFC3339))
				fmt.Println("Run 'omdr auth login' to re-authenticate")
				return nil
			}
			fmt.Println("Authenticated")
			fmt.Printf("Token expires: %s\n", expTime.Format(time.RFC3339))
		} else {
			fmt.Println("Authenticated")
		}

		// Try to get username from API
		apiURL := viper.GetString("api_url")
		if apiURL == "" {
			apiURL = "http://localhost:8080"
		}

		apiClient := client.NewClient(apiURL)
		apiClient.SetToken(token)

		var user struct {
			Username string `json:"username"`
		}
		if err := apiClient.Get("/api/v1/users/me", &user); err == nil {
			fmt.Printf("Username: %s\n", user.Username)
		}

		return nil
	},
}

func startCallbackServer(tokenChan chan string, errChan chan error) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			errChan <- fmt.Errorf("no token received")
			http.Error(w, "Authentication failed", http.StatusBadRequest)
			return
		}

		tokenChan <- token

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<html>
			<head><title>Authentication Successful</title></head>
			<body>
				<h1>Authentication Successful!</h1>
				<p>You can close this window and return to the terminal.</p>
			</body>
			</html>
		`)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", callbackPort),
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	return srv
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}
