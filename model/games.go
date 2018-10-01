
package model

import (
  "crypto/rand"
  "database/sql"
  "encoding/base64"
  "fmt"
  "time"
  "github.com/go-sql-driver/mysql"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

type Game struct {
  Id string
  Game_key string
  Created_at time.Time
  Updated_at time.Time
  First_block string
  Last_block string
  Started_at mysql.NullTime
  Round_ends_at mysql.NullTime
  Current_round int
}

type GamePlayer struct {
  Game_id string
  Rank int
  Team_id string
  Team_player int
  Created_at time.Time
  Updated_at time.Time
  Commands string
}

func (m *Model) CreateGame(firstBlock string) (string, error) {
  var err error
  gameKey, err := generateKey()
  if err != nil { return "", errors.Wrap(err, 0) }
  _, err = m.db.Exec(
    `INSERT INTO games (game_key, first_block, last_block, current_round)
     VALUES (?, ?, ?, 0)`, gameKey, firstBlock, firstBlock)
  if err != nil { return "", errors.Wrap(err, 0) }
  return gameKey, nil
}

func (m *Model) ViewGame(gameKey string) (j.Value, error) {
  game, err := m.loadGame(gameKey, NullFacet)
  if err != nil { return j.Null, err }
  if game == nil {
    return j.Null, nil
  }
  return viewGame(game), nil
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

func (m *Model) loadGamePlayers (gameKey string, teamId string, f Facets) ([]GamePlayer, error) {
  var err error
  rows, err := m.db.Queryx(
    `SELECT gp.* FROM game_players gp
      INNER JOIN games g ON gp.game_id = g.id
      WHERE g.game_key = ? AND gp.team_id = ?
      ORDER BY rank`,
    gameKey, teamId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  defer rows.Close()
  var items []GamePlayer
  for rows.Next() {
    item, err := m.loadGamePlayerRow(rows, f)
    if err != nil { return nil, err }
    items = append(items, *item)
  }
  return items, nil
}

func (m *Model) getGameCommands (gameId string) ([]GamePlayer, error) {
  var err error
  rows, err := m.db.Queryx(
    `SELECT * FROM game_players gp WHERE game_id = ? ORDER BY rank`, gameId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  defer rows.Close()
  var items []GamePlayer
  for rows.Next() {
    item, err := m.loadGamePlayerRow(rows, NullFacet)
    if err != nil { return nil, err }
    items = append(items, *item)
  }
  return items, nil
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
    m.Add(fmt.Sprintf("games %s", res.Id), viewGame(&res))
  }
  return &res, nil
}

func (m *Model) loadGamePlayerRow(row IRow, f Facets) (*GamePlayer, error) {
  var res GamePlayer
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  if f.Base {
    view := j.Object()
    view.Prop("rank", j.Int(res.Rank))
    view.Prop("teamPlayer", j.Int(res.Team_player))
    timeProp(view, "createdAt", res.Created_at)
    timeProp(view, "updatedAt", res.Updated_at)
    view.Prop("commands", j.String(res.Commands))
  }
  return &res, nil
}

func generateKey() (string, error) {
  bs := make([]byte, 32, 32)
  _, err := rand.Read(bs)
  if err != nil { return "", err }
  return base64.RawURLEncoding.EncodeToString(bs[:]), nil
}

func viewGame(game *Game) j.IObject {
  view := j.Object()
  view.Prop("key", j.String(game.Game_key))
  timeProp(view, "createdAt", game.Created_at)
  timeProp(view, "updatedAt", game.Updated_at)
  view.Prop("firstBlock", j.String(game.First_block))
  view.Prop("lastBlock", j.String(game.Last_block))
  nullTimeProp(view, "startedAt", game.Started_at)
  nullTimeProp(view, "roundEndsAt", game.Round_ends_at)
  view.Prop("currentRound", j.Int(game.Current_round))
  return view
}