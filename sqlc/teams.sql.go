// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: teams.sql

package sqlc

import (
	"context"
	"strings"
)

const addMatchTeam = `-- name: AddMatchTeam :exec
INSERT INTO teams (
    channel_id,
    role_id
) VALUES (
    ?1,
    ?2
)
`

type AddMatchTeamParams struct {
	ChannelID string `db:"channel_id"`
	RoleID    string `db:"role_id"`
}

func (q *Queries) AddMatchTeam(ctx context.Context, arg AddMatchTeamParams) error {
	_, err := q.exec(ctx, q.addMatchTeamStmt, addMatchTeam, arg.ChannelID, arg.RoleID)
	return err
}

const addMatchTeamResults = `-- name: AddMatchTeamResults :exec
UPDATE teams
SET
    score = ?1,
    screenshot = ?2,
    demo = ?3
WHERE channel_id = ?4
AND role_id = ?5
`

type AddMatchTeamResultsParams struct {
	Score      int64  `db:"score"`
	Screenshot []byte `db:"screenshot"`
	Demo       []byte `db:"demo"`
	ChannelID  string `db:"channel_id"`
	RoleID     string `db:"role_id"`
}

func (q *Queries) AddMatchTeamResults(ctx context.Context, arg AddMatchTeamResultsParams) error {
	_, err := q.exec(ctx, q.addMatchTeamResultsStmt, addMatchTeamResults,
		arg.Score,
		arg.Screenshot,
		arg.Demo,
		arg.ChannelID,
		arg.RoleID,
	)
	return err
}

const decreaseMatchTeamConfirmedParticipants = `-- name: DecreaseMatchTeamConfirmedParticipants :exec
UPDATE teams
SET confirmed_participants = confirmed_participants - 1
WHERE channel_id = ?1
AND role_id = ?2
AND confirmed_participants > 0
`

type DecreaseMatchTeamConfirmedParticipantsParams struct {
	ChannelID string `db:"channel_id"`
	RoleID    string `db:"role_id"`
}

func (q *Queries) DecreaseMatchTeamConfirmedParticipants(ctx context.Context, arg DecreaseMatchTeamConfirmedParticipantsParams) error {
	_, err := q.exec(ctx, q.decreaseMatchTeamConfirmedParticipantsStmt, decreaseMatchTeamConfirmedParticipants, arg.ChannelID, arg.RoleID)
	return err
}

const deleteAllMatchTeams = `-- name: DeleteAllMatchTeams :exec
DELETE FROM teams
WHERE channel_id = ?1
`

func (q *Queries) DeleteAllMatchTeams(ctx context.Context, channelID string) error {
	_, err := q.exec(ctx, q.deleteAllMatchTeamsStmt, deleteAllMatchTeams, channelID)
	return err
}

const deleteMatchTeam = `-- name: DeleteMatchTeam :exec
DELETE FROM teams
WHERE channel_id = ?1
AND role_id = ?2
`

type DeleteMatchTeamParams struct {
	ChannelID string `db:"channel_id"`
	RoleID    string `db:"role_id"`
}

func (q *Queries) DeleteMatchTeam(ctx context.Context, arg DeleteMatchTeamParams) error {
	_, err := q.exec(ctx, q.deleteMatchTeamStmt, deleteMatchTeam, arg.ChannelID, arg.RoleID)
	return err
}

const getMatchTeam = `-- name: GetMatchTeam :one
SELECT
    channel_id,
    role_id,
    confirmed_participants
FROM teams
WHERE channel_id = ?1
AND role_id = ?2
`

type GetMatchTeamParams struct {
	ChannelID string `db:"channel_id"`
	RoleID    string `db:"role_id"`
}

type GetMatchTeamRow struct {
	ChannelID             string `db:"channel_id"`
	RoleID                string `db:"role_id"`
	ConfirmedParticipants int64  `db:"confirmed_participants"`
}

func (q *Queries) GetMatchTeam(ctx context.Context, arg GetMatchTeamParams) (GetMatchTeamRow, error) {
	row := q.queryRow(ctx, q.getMatchTeamStmt, getMatchTeam, arg.ChannelID, arg.RoleID)
	var i GetMatchTeamRow
	err := row.Scan(&i.ChannelID, &i.RoleID, &i.ConfirmedParticipants)
	return i, err
}

const getMatchTeamByRoles = `-- name: GetMatchTeamByRoles :many
SELECT
    channel_id,
    role_id,
    confirmed_participants
FROM teams
WHERE channel_id = ?1
AND role_id IN (/*SLICE::role_ids*/?)
ORDER BY role_id
`

type GetMatchTeamByRolesParams struct {
	ChannelID string   `db:"channel_id"`
	RoleIds   []string `db:":role_ids"`
}

type GetMatchTeamByRolesRow struct {
	ChannelID             string `db:"channel_id"`
	RoleID                string `db:"role_id"`
	ConfirmedParticipants int64  `db:"confirmed_participants"`
}

func (q *Queries) GetMatchTeamByRoles(ctx context.Context, arg GetMatchTeamByRolesParams) ([]GetMatchTeamByRolesRow, error) {
	query := getMatchTeamByRoles
	var queryParams []interface{}
	queryParams = append(queryParams, arg.ChannelID)
	if len(arg.RoleIds) > 0 {
		for _, v := range arg.RoleIds {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE::role_ids*/?", strings.Repeat(",?", len(arg.RoleIds))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE::role_ids*/?", "NULL", 1)
	}
	rows, err := q.query(ctx, nil, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetMatchTeamByRolesRow{}
	for rows.Next() {
		var i GetMatchTeamByRolesRow
		if err := rows.Scan(&i.ChannelID, &i.RoleID, &i.ConfirmedParticipants); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const increaseMatchTeamConfirmedParticipants = `-- name: IncreaseMatchTeamConfirmedParticipants :exec
UPDATE teams
SET confirmed_participants = confirmed_participants + 1
WHERE channel_id = ?1
AND role_id = ?2
`

type IncreaseMatchTeamConfirmedParticipantsParams struct {
	ChannelID string `db:"channel_id"`
	RoleID    string `db:"role_id"`
}

func (q *Queries) IncreaseMatchTeamConfirmedParticipants(ctx context.Context, arg IncreaseMatchTeamConfirmedParticipantsParams) error {
	_, err := q.exec(ctx, q.increaseMatchTeamConfirmedParticipantsStmt, increaseMatchTeamConfirmedParticipants, arg.ChannelID, arg.RoleID)
	return err
}

const listMatchTeams = `-- name: ListMatchTeams :many
SELECT
    channel_id,
    role_id,
    confirmed_participants
FROM teams
WHERE channel_id = ?1
ORDER BY role_id
`

type ListMatchTeamsRow struct {
	ChannelID             string `db:"channel_id"`
	RoleID                string `db:"role_id"`
	ConfirmedParticipants int64  `db:"confirmed_participants"`
}

func (q *Queries) ListMatchTeams(ctx context.Context, channelID string) ([]ListMatchTeamsRow, error) {
	rows, err := q.query(ctx, q.listMatchTeamsStmt, listMatchTeams, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListMatchTeamsRow{}
	for rows.Next() {
		var i ListMatchTeamsRow
		if err := rows.Scan(&i.ChannelID, &i.RoleID, &i.ConfirmedParticipants); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
