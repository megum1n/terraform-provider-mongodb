package mongodb

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	createRoleCmd = "createRole"
	getRoleCmd    = "rolesInfo"
	updateRoleCmd = "updateRole"
	deleteRoleCmd = "dropRole"
)

func (c *Client) UpsertRole(ctx context.Context, role *Role) (*Role, error) {
	tflog.Debug(ctx, "UpsertRole", map[string]any{
		"name":     role.Name,
		"database": role.Database,
	})

	var cmd string

	_, err := c.GetRole(ctx, &GetRoleOptions{
		Name:     role.Name,
		Database: role.Database,
	})

	switch {
	case errors.As(err, &NotFoundError{}):
		cmd = createRoleCmd
	case err == nil:
		cmd = updateRoleCmd
	default:
		return nil, err
	}

	command := bson.D{
		{Key: cmd, Value: role.Name},
		{Key: "privileges", Value: role.Privileges.toBson()},
		// Roles field is required, but empty array is fine
		{Key: "roles", Value: role.Roles.toBson()},
	}

	response := c.mongo.Database(role.Database).RunCommand(ctx, command)

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

	role, err = c.GetRole(ctx, &GetRoleOptions{
		Name:     role.Name,
		Database: role.Database,
	})
	if err != nil {
		return nil, err
	}

	return role, nil
}

type GetRoleOptions struct {
	Name     string
	Database string
}

type getRoleResult struct {
	Ok    int    `bson:"ok"`
	Roles []Role `bson:"roles"`
}

func (c *Client) GetRole(ctx context.Context, options *GetRoleOptions) (*Role, error) {
	tflog.Debug(ctx, "GetRole", map[string]any{
		"name":     options.Name,
		"database": options.Database,
	})

	command := bson.D{
		{Key: getRoleCmd, Value: options.Name},
		{Key: "showPrivileges", Value: true},
	}

	response := c.mongo.Database(options.Database).RunCommand(ctx, command)

	err := response.Err()
	if err != nil {
		return nil, err
	}

	var result getRoleResult

	err = response.Decode(&result)
	if err != nil {
		return nil, err
	}

	if result.Ok != 1 {
		return nil, FailedCommandError{getRoleCmd}
	}

	roleCount := len(result.Roles)

	switch {
	case roleCount == 0:
		return nil, NotFoundError{options.Name, "role"}
	case roleCount > 1:
		return nil, TooManyError{"role"}
	}

	return &result.Roles[0], nil
}

type DeleteRoleOptions struct {
	Name     string
	Database string
}

func (c *Client) DeleteRole(ctx context.Context, options *DeleteRoleOptions) error {
	tflog.Debug(ctx, "DeleteRole", map[string]interface{}{
		"name":     options.Name,
		"database": options.Database,
	})

	command := bson.D{
		{Key: deleteRoleCmd, Value: options.Name},
	}

	response := c.mongo.Database(options.Database).RunCommand(ctx, command)

	err := response.Err()
	if err != nil {
		return err
	}

	var result Result

	err = response.Decode(&result)
	if err != nil {
		return err
	}

	if result.Ok != 1 {
		return FailedCommandError{deleteRoleCmd}
	}

	return nil
}
