package skill

import "testing"

func TestValidate(t *testing.T) {
	valid := func() Skill {
		return Skill{
			Name:        "python",
			Description: "Run Python 3 scripts.",
			Category:    "stack",
			Runtime:     "python3",
		}
	}

	tests := []struct {
		name    string
		mutate  func(*Skill)
		wantErr bool
	}{
		{name: "valid", mutate: func(*Skill) {}, wantErr: false},
		{name: "empty name", mutate: func(s *Skill) { s.Name = "" }, wantErr: true},
		{name: "empty runtime", mutate: func(s *Skill) { s.Runtime = " " }, wantErr: true},
		{name: "valid category role", mutate: func(s *Skill) { s.Category = "role" }, wantErr: false},
		{name: "valid category stack", mutate: func(s *Skill) { s.Category = "stack" }, wantErr: false},
		{name: "valid category process", mutate: func(s *Skill) { s.Category = "process" }, wantErr: false},
		{name: "empty category ok", mutate: func(s *Skill) { s.Category = "" }, wantErr: false},
		{name: "bad category", mutate: func(s *Skill) { s.Category = "other" }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := valid()
			tt.mutate(&s)
			err := s.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
