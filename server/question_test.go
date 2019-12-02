package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandomQuestion(t *testing.T) {
	question, err := RandomQuestion()
	require.NoError(t, err)
	require.True(t, question.IsCorrectAnswer(question.Answers[0].Answer))
}
