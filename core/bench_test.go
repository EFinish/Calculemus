package core

import (
	"encoding/json"
	"os"
	"testing"
)

func BenchmarkEvaluateChildBall(b *testing.B) {
	data, _ := os.ReadFile("../examples/child-and-ball.json")
	var u Universe
	_ = json.Unmarshal(data, &u)
	for s := range u.Statements {
		if u.Statements[s].ID == "s1" {
			u.Statements[s].Predicate = "green"
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Evaluate(&u); err != nil {
			b.Fatal(err)
		}
	}
}
