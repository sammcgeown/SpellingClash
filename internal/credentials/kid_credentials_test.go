package credentials

import "testing"

func TestGenerateKidPassword(t *testing.T) {
	tests := []struct {
		name        string
		iterations  int
		minLength   int
		maxLength   int
		shouldMatch bool
	}{
		{
			name:       "generates password of correct length",
			iterations: 100,
			minLength:  4,
			maxLength:  4,
		},
		{
			name:        "generates unique passwords",
			iterations:  10,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passwords := make(map[string]bool)
			for i := 0; i < tt.iterations; i++ {
				password, _ := GenerateKidPassword()

				// Check length
				if len(password) < tt.minLength || len(password) > tt.maxLength {
					t.Errorf("password length %d not in range [%d, %d]", len(password), tt.minLength, tt.maxLength)
				}

				// Check uniqueness
				if !tt.shouldMatch {
					if passwords[password] {
						t.Errorf("duplicate password generated: %s", password)
					}
					passwords[password] = true
				}
			}
		})
	}
}
