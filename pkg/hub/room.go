package hub

import (
	"context"
	"fmt"
	"rps_website/pkg/assert"
	"strconv"
)

// Look, I don't know what happened here either, and I'm not touching it ever again...

type Room struct {
	players [2]*Client
}

type Buf []byte

func (buf *Buf) Write(p []byte) (n int, err error) {
	*buf = p
	return len(p), nil
}


func (room *Room) mediate(room_num int) {
	fmt.Printf("Mediating game in Room #%d", room_num + 1)
	player_1 := room.players[0]
	player_1.in_match = true
	player_2 := room.players[1]
	player_2.in_match = true

	assert.Assert_ne(
		player_1.connection.RemoteAddr(),
		player_2.connection.RemoteAddr(),
		"ADDRESSES SHOULD NOT BE EQUAL",
	)

	var game_screen1 Buf
	play_screen(player_2.player_name).Render(context.Background(), &game_screen1)
	player_1.message_channel <- game_screen1

	// I'm using this because there's some kind of race condition with the
	// server writing to the clients at the same time, causing them to have the
	// same game screen where both players will see "Playing against {name}"
	// where `name` is the same for both players, which shouldn't be the case.
	// go run -race doesn't pick up this race condition and I'm not sure how
	// else to solve it...
	<-player_1.message_channel

	var game_screen2 Buf
	play_screen(player_1.player_name).Render(context.Background(), &game_screen2)
	player_2.message_channel <- game_screen2

	<-player_2.message_channel

	p1_action, p2_action := 0, 0
	p1_action_str, p2_action_str := "", ""

	// since we don't know which order messages will come in from each player,
	// we pass both players into the function and select on who sends a message
	// first
	room.handle_single_message( // get first action
		player_1, player_2,
		room_num,
		&p1_action, &p2_action,
		&p1_action_str, &p2_action_str,
		)

	room.handle_single_message( // get second action
		player_1, player_2,
		room_num,
		&p1_action, &p2_action,
		&p1_action_str, &p2_action_str,
		)

	var result int = 0

	if p1_action != p2_action {
		result = result_map[p1_action][p2_action]
	}

	assert.Assert(result == 0 || result == 1 || result == 2, fmt.Sprintf("unexpected game result: %d", result))

	switch result {
	case 0:
		send_result_screen(player_1, player_2, p1_action_str, p2_action_str, true)
	case 1:
		send_result_screen(player_1, player_2, p1_action_str, p2_action_str, false)
	case 2:
		send_result_screen(player_2, player_1, p2_action_str, p1_action_str, false)
	}

	room.remove_both_players(player_1, player_2, room_num)
}

func (room *Room) remove_both_players(player_1, player_2 *Client, room_num int) {
	<-player_1.message_channel
	<-player_2.message_channel
	player_1.in_match = false
	player_2.in_match = false

	if room.players[0] != nil {
		room.players[0] = nil
		fmt.Printf("Removed player from Room #%d, Spot #1\n", room_num+1)
	}
	if room.players[1] != nil {
		room.players[1] = nil
		fmt.Printf("Removed player from Room #%d, Spot #2\n", room_num+1)
	}
}

func (room *Room) handle_single_message(
	player_1, player_2 *Client,
	room_num int,
	p1_action, p2_action *int,
	p1_action_str, p2_action_str *string,
) {
	select {
	case client_message := <-player_1.message_channel:
		err := room.handle_player_leave(player_1, player_2, client_message, 0, room_num)
		if err != nil {
			player_1.send_error_screen()
			room.remove_both_players(player_1, player_2, room_num)
			println("An error occured processing p1's action: ", err.Error())
		}
		action, action_str, err := room.handle_player_action_input(
			client_message,
			player_1, // main player
			player_2, // other player
			)

		if err != nil {
			player_1.send_error_screen()
			room.remove_both_players(player_1, player_2, room_num)
			println("An error occured processing p1's action: ", err.Error())
		}
		*p1_action = action
		*p1_action_str = action_str

	case client_message := <-player_2.message_channel:
		err := room.handle_player_leave(player_2, player_1, client_message, 0, room_num)
		if err != nil {
			player_1.send_error_screen()
			room.remove_both_players(player_1, player_2, room_num)
			println("An error occured processing p1's action: ", err.Error())
		}
		action, action_str, err := room.handle_player_action_input(
			client_message,
			player_2, // main player
			player_1, // other player
			)

		if err != nil {
			player_2.send_error_screen()
			room.remove_both_players(player_1, player_2, room_num)
			println("An error occured processing p2's action: ", err.Error())
		}
		*p2_action = action
		*p2_action_str = action_str
	}
}

const (
	ROCK = iota + 1
	PAPER
	SCISSORS
)

var result_map = map[int]map[int]int{
	ROCK: {
		SCISSORS: 1,
		PAPER:    2,
	},
	PAPER: {
		ROCK:     1,
		SCISSORS: 2,
	},
	SCISSORS: {
		PAPER: 1,
		ROCK:  2,
	},
}

func send_result_screen(
	main_player, other_player *Client,
	main_player_action, other_player_action string,
	tie bool,
) {
	var main_position, other_position string
	if tie {
		main_position = "tied"
		other_position = "tied"
	} else {
		main_position = "won"
		other_position = "lost"
	}
	var result_screen1 Buf
	other_action_chose(other_player_action, other_player.player_name, main_position).Render(context.Background(), &result_screen1)
	main_player.message_channel <- result_screen1

	<-main_player.message_channel

	var result_screen2 Buf
	other_action_chose(main_player_action, main_player.player_name, other_position).Render(context.Background(), &result_screen2)
	other_player.message_channel <- result_screen2

	<-other_player.message_channel
}

func (room *Room) handle_player_leave(
	remaning_player, departed_player *Client,
	client_message []byte,
	player_num, room_num int,
) error {
	if string(client_message) == "left" {
		remaning_player.send_player_left_screen(departed_player.player_name)
		room.players[player_num] = nil
		remaning_player.in_match = false

		if player_num == 0 {
			room.players[1] = nil
			fmt.Printf("Removed player from Room #%d, Spot #2\n", room_num+1)
		} else {
			room.players[0] = nil
			fmt.Printf("Removed player from Room #%d, Spot #1\n", room_num+1)
		}
		departed_player.in_match = false

		// this error break/continue thing is probably not the best way of
		// doing this...
		return fmt.Errorf("break")
	}
	return nil
}

func (room *Room) handle_player_action_input(client_message []byte,
	main_player, other_player *Client,
) (int, string, error) {

	player_input, err := strconv.Atoi(string(client_message))
	if err != nil {
		return -1, "", fmt.Errorf("couldn't parse player action: %s", err.Error())
	}

	action, err := convert_and_verify_action(player_input)
	if err != nil {
		return -1, "", fmt.Errorf("couldn't parse player action: %s", err.Error())
	}

	main_player.send_action_confirm_screen(action, other_player.player_name)

	return player_input, action, nil
}

func convert_and_verify_action(action int) (string, error) {
	switch action {
	case ROCK:
		return "rock", nil
	case PAPER:
		return "paper", nil
	case SCISSORS:
		return "scissors", nil
	default:
		return "", fmt.Errorf("%d is not a valid action", action)
	}
}
