package agent

import "testing"

func TestValidate(t *testing.T) {
	valid := func() Agent {
		return Agent{
			Name:        "Code Gardener",
			Prompt:      "Refactor legacy code for readability.",
			Provider:    "deepseek",
			Model:       "deepseek-v4-flash",
			Temperature: 0.2,
		}
	}

	tests := []struct {
		name    string
		mutate  func(*Agent)
		wantErr bool
	}{
		{name: "valid", mutate: func(*Agent) {}, wantErr: false},
		{name: "empty name", mutate: func(a *Agent) { a.Name = "  " }, wantErr: true},
		{name: "empty prompt", mutate: func(a *Agent) { a.Prompt = "" }, wantErr: true},
		{name: "empty provider", mutate: func(a *Agent) { a.Provider = "" }, wantErr: true},
		{name: "empty model", mutate: func(a *Agent) { a.Model = "" }, wantErr: true},
		{name: "temp below zero", mutate: func(a *Agent) { a.Temperature = -0.1 }, wantErr: true},
		{name: "temp above one", mutate: func(a *Agent) { a.Temperature = 1.1 }, wantErr: true},
		{name: "temp at boundary one", mutate: func(a *Agent) { a.Temperature = 1.0 }, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := valid()
			tt.mutate(&a)
			err := a.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
