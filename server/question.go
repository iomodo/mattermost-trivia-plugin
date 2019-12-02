package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// Question describes questions
type Question struct {
	ID       string
	Question string
	Category string
	Points   int
	Answers  []Answer
}

// Answer describes single answer
type Answer struct {
	Answer  string
	Correct bool
}

// IsCorrectAnswer returns true if answer is the correct one
func (q *Question) IsCorrectAnswer(answer string) bool {
	answer = strings.ToLower(answer)
	for _, ans := range q.Answers {
		if strings.Contains(answer, strings.ToLower(ans.Answer)) && ans.Correct {
			return true
		}
	}
	return false
}

// GetAnswer returns answer or a list of answers
func (q *Question) GetAnswer() string {
	res := ""
	for _, ans := range q.Answers {
		if ans.Correct {
			res += ans.Answer + " "
		}
	}
	return res
}

type jServiceQuestion struct {
	ID           int              `json:"id"`
	Answer       string           `json:"answer"`
	Question     string           `json:"question"`
	Value        int              `json:"value"`
	Airdate      string           `json:"airdate"`
	CreatedAt    string           `json:"created_at"`
	UpdatedAt    string           `json:"updated_at"`
	CategoryID   int              `json:"category_id"`
	GameID       int              `json:"game_id"`
	InvalidCount int              `json:"invalid_count"`
	Category     jServiceCategory `json:"category"`
}

type jServiceCategory struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	CluesCount int    `json:"clues_count"`
}

// RandomQuestion returns random question from jservice.io
func RandomQuestion() (*Question, error) {
	url := "http://jservice.io/api/random"
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "can't get random question from %s", url)
	}

	questions := make([]*jServiceQuestion, 0)
	err = json.NewDecoder(resp.Body).Decode(&questions)
	if err != nil {
		return nil, errors.Wrapf(err, "can't get decode response from %s", url)
	}
	if len(questions) != 1 {
		return nil, errors.Wrapf(err, "unknown response %v", questions)
	}
	result := &Question{
		ID:       fmt.Sprintf("jservice_%d", questions[0].ID),
		Question: questions[0].Question,
		Category: questions[0].Category.Title,
		Points:   questions[0].Value,
		Answers:  []Answer{{Answer: questions[0].Answer, Correct: true}},
	}
	return result, nil
}
