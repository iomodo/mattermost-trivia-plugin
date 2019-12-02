package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	trigger               = "trivia"
	randomQuestionAcrtion = "q"
	answerQuestion        = "a"
	topPlayers            = "top"

	displayName = "Trivia"
	desc        = "Mattermost Trivia Plugin"

	topPlayersCount = 10
)

const commandHelp = `
* |/trivia q| - Outputs random trivia question you have to answer in 60 seconds
* |/trivia a _answer_| - Answer to the posted question
* |/trivia top| - Outputs scores of top players as well as your own score
`

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          trigger,
		DisplayName:      displayName,
		Description:      desc,
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: q, a, top, help",
		AutoCompleteHint: "[command]",
	}
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		Message:   text,
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)
}

func (p *Plugin) helpResponse(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	text := "###### " + desc + " - Slash Command Help\n" + strings.Replace(commandHelp, "|", "`", -1)
	p.postCommandResponse(args, text)
	return &model.CommandResponse{}, nil
}

func appError(message string, err error) *model.AppError {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	return model.NewAppError("Trivia Plugin", message, nil, errorMessage, http.StatusBadRequest)
}

func (p *Plugin) askRandomQuestion(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	p.dataLock.RLock()
	if _, ok := p.userData[args.UserId]; ok {
		p.postCommandResponse(args, "Answer the previous question first")
		p.dataLock.RUnlock()
		return &model.CommandResponse{}, nil
	}
	p.dataLock.RUnlock()

	q, err := RandomQuestion()
	if err != nil {
		return nil, appError("Can't get random question", err)
	}
	text := fmt.Sprintf("Category : %s \n Points: %d \n Question: %s \n", q.Category, q.Points, q.Question)
	p.postCommandResponse(args, text)
	timer := time.NewTimer(60 * time.Second)
	data := &UserData{
		Question: q,
		Timer:    timer,
	}
	p.dataLock.Lock()
	p.userData[args.UserId] = data
	go func(data *UserData) {
		<-data.Timer.C
		p.postCommandResponse(args, fmt.Sprintf("Too late! Correct answer was %s. Try again!", data.Question.GetAnswer()))
		delete(p.userData, args.UserId)
	}(data)
	p.dataLock.Unlock()
	return &model.CommandResponse{}, nil
}

func (p *Plugin) answerQuestion(args *model.CommandArgs, arg string) (*model.CommandResponse, *model.AppError) {
	p.dataLock.RLock()
	defer p.dataLock.RUnlock()
	if data, ok := p.userData[args.UserId]; ok {
		defer delete(p.userData, args.UserId)
		if data.Timer.Stop() {
			if data.Question.IsCorrectAnswer(arg) {
				p.postCommandResponse(args, "Correct!")
				userScores := make(map[string]int)
				ok, err := p.Helpers.KVGetJSON(trigger, &userScores)
				if err == nil && ok {
					if _, ok = userScores[args.UserId]; ok {
						userScores[args.UserId] += data.Question.Points
					} else {
						userScores[args.UserId] = data.Question.Points
					}
					p.Helpers.KVSetJSON(trigger, &userScores)
					p.postCommandResponse(args, fmt.Sprintf("Your score is %d", userScores[args.UserId]))
				}
				return &model.CommandResponse{}, nil
			}
			p.postCommandResponse(args, fmt.Sprintf("Incorrect! Correct answer is %s", data.Question.GetAnswer()))
			return &model.CommandResponse{}, nil
		}
	} else {
		p.postCommandResponse(args, "Take a question first")
		return &model.CommandResponse{}, nil
	}
	return &model.CommandResponse{}, nil
}

func (p *Plugin) outputTopPlayers(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	userScores := make(map[string]int)
	ok, err := p.Helpers.KVGetJSON(trigger, &userScores)
	if err != nil && !ok {
		return &model.CommandResponse{}, appError("Can't output top players", err)
	}
	sortedScores := sortByValue(userScores)
	if len(sortedScores) > topPlayersCount {
		sortedScores = sortedScores[:topPlayersCount]
	}
	text := "Top players are:\n"
	for i, pair := range sortedScores {
		user, err := p.API.GetUser(pair.Key)
		if err == nil {
			text += fmt.Sprintf("%d. @%s - %d\n", i+1, user.Username, pair.Value)
		} else {
			text += fmt.Sprintf("%d. %s - %d\n", i+1, "unknown user", pair.Value)
		}
	}
	p.postCommandResponse(args, text)
	return &model.CommandResponse{}, nil
}

// ExecuteCommand executes a command that has been previously registered via the RegisterCommand API.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	if len(split) == 0 {
		return nil, nil
	}
	command := split[0]
	action := ""
	arg := ""
	if len(split) > 1 {
		action = split[1]
	}
	if len(split) > 2 {
		arg = strings.Join(split[2:], " ")
	}
	if command != "/"+trigger {
		return &model.CommandResponse{}, nil
	}
	switch action {
	case "":
		return p.helpResponse(args)
	case "help":
		return p.helpResponse(args)
	case randomQuestionAcrtion:
		return p.askRandomQuestion(args)
	case answerQuestion:
		return p.answerQuestion(args, arg)
	case topPlayers:
		return p.outputTopPlayers(args)
	}

	return &model.CommandResponse{}, nil
}
