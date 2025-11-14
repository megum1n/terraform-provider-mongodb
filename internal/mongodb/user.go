package mongodb

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	createUserCmd = "createUser"
	getUserCmd    = "usersInfo"
	updateUserCmr = "updateUser"
	deleteUserCmd = "dropUser"
)

func (c *Client) UpsertUser(ctx context.Context, user *User) (*User, error) {
	tflog.Debug(ctx, "UpsertUser", map[string]interface{}{
		"username": user.Username,
		"db":       user.Database,
	})

	var cmd string

	getUserOptions := &GetUserOptions{
		Username: user.Username,
		Database: user.Database,
	}
	_, err := c.GetUser(ctx, getUserOptions)

	switch {
	case errors.As(err, &NotFoundError{}):
		cmd = createUserCmd
	case err == nil:
		cmd = updateUserCmr
	default:
		return nil, err
	}

	command := bson.D{
		{Key: cmd, Value: user.Username},
		// Roles field is required, but empty array is fine
		{Key: "roles", Value: user.Roles.toBson()},
	}

	if user.Password != "" {
		command = append(command, bson.E{Key: "pwd", Value: user.Password})
	}

	if len(user.Mechanisms) > 0 {
		command = append(command, bson.E{Key: "mechanisms", Value: user.Mechanisms})
	}

	response := c.mongo.Database(user.Database).RunCommand(ctx, command)

	err = response.Err()
	if err != nil {
		return nil, err
	}

	result := &Result{}

	err = response.Decode(result)
	if err != nil {
		return nil, err
	}

	if result.Ok != 1 {
		return nil, FailedCommandError{cmd}
	}

	user, err = c.GetUser(ctx, getUserOptions)
	if err != nil {
		return nil, err
	}

	return user, nil
}

type GetUserOptions struct {
	Username string
	Database string
}

type getUsersResult struct {
	Ok    int    `bson:"ok"`
	Users []User `bson:"users"`
}

func (c *Client) GetUser(ctx context.Context, options *GetUserOptions) (*User, error) {
	tflog.Debug(ctx, "GetUser", map[string]interface{}{
		"username": options.Username,
		"db":       options.Database,
	})

	command := bson.D{
		{Key: getUserCmd, Value: options.Username},
	}

	response := c.mongo.Database(options.Database).RunCommand(ctx, command)
	if err := response.Err(); err != nil {
		return nil, err
	}

	var result getUsersResult

	err := response.Decode(&result)
	if err != nil {
		return nil, err
	}

	if result.Ok != 1 {
		return nil, FailedCommandError{getUserCmd}
	}

	userCount := len(result.Users)

	switch {
	case userCount == 0:
		return nil, NotFoundError{}
	case userCount > 1:
		return nil, TooManyError{t: "user"}
	}

	return &result.Users[0], nil
}

type DeleteUserOptions struct {
	Username string
	Database string
}

func (c *Client) DeleteUser(ctx context.Context, options *DeleteUserOptions) error {
	tflog.Debug(ctx, "DeleteUser", map[string]interface{}{
		"username": options.Username,
		"db":       options.Database,
	})

	command := bson.D{
		{Key: deleteUserCmd, Value: options.Username},
	}

	response := c.mongo.Database(options.Database).RunCommand(ctx, command)
	if err := response.Err(); err != nil {
		return err
	}

	result := Result{}

	err := response.Decode(&result)
	if err != nil {
		return err
	}

	if result.Ok != 1 {
		return FailedCommandError{deleteUserCmd}
	}

	return nil
}
