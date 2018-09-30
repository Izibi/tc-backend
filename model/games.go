
package model

import (
  "database/sql"
  "time"
  "github.com/go-sql-driver/mysql"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

type Game struct {
  Game_key string
  Created_at time.Time
  Updated_at time.Time
  Started_at mysql.NullTime
  Round_ends_at mysql.NullTime
  First_block string
  Current_block string
  Current_round int
  Game_params string /*json*/
  Task_params string /*json*/
}

func (m *Model) addGame(key string, gameParams string, taskParams string, firstBlock string) error {
  var err error
  _, err = m.db.Exec(
    `INSERT INTO games (game_key, first_block, current_block, game_params, task_params)
     VALUES (?, ?, ?, ?, ?)`, key, firstBlock, firstBlock, gameParams, taskParams)
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

func (m *Model) addPlayerToGame (gameKey string, teamId string, teamPlayer int, commands string) error {
  var err error
  _, err = m.db.Exec(
    `INSERT INTO game_players (game_id, rank, team_id, team_player, commands)
      SELECT g.id, 1 + COUNT(gp.rank), ?, ?, ?
      FROM games g, game_players gp
      WHERE g.game_key = ?
      AND gp.game_id = g.id`,
    teamId, teamPlayer, commands, gameKey)
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

func (m *Model) setPlayerCommands (gameKey string, teamId string, teamPlayer int, commands string) error {
  var err error
  _, err = m.db.Exec(
    `UPDATE game_players gp
      INNER JOIN games g ON gp.game_id = g.id
      SET commands = ?
      WHERE g.game_key = ?
      AND gp.team_id = ?
      AND gp.team_player = ?`,
    commands, gameKey, teamId, teamPlayer)
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

func (m *Model) loadGame(key string, f Facets) (*Game, error) {
  return m.loadGameRow(m.db.QueryRowx(
    `SELECT * FROM games WHERE game_key = ?`, key), f)
}

func (m *Model) loadGameRow(row IRow, f Facets) (*Game, error) {
  var res Game
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  if f.Base {
    view := j.Object()
    view.Prop("key", j.String(res.Game_key))
    timeProp(view, "createdAt", res.Created_at)
    timeProp(view, "updatedAt", res.Updated_at)
    nullTimeProp(view, "startedAt", res.Started_at)
    nullTimeProp(view, "roundEndsAt", res.Round_ends_at)
    view.Prop("firstBlock", j.String(res.First_block))
    view.Prop("currentBlock", j.String(res.Current_block))
  }
  return &res, nil
}
