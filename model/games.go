
package model

import (
  "crypto/rand"
  "database/sql"
  "encoding/base64"
  "encoding/json"
  "fmt"
  "time"
  "github.com/go-sql-driver/mysql"
  "github.com/go-errors/errors"
  ji "github.com/json-iterator/go"
  j "tezos-contests.izibi.com/backend/jase"
)

type Game struct {
  Id string
  Game_key string
  Created_at time.Time
  Updated_at time.Time
  Owner_id string
  First_block string
  Last_block string
  Started_at mysql.NullTime
  Round_ends_at mysql.NullTime
  Current_round uint
  Locked bool
  Next_block_commands []byte
}

type GamePlayer struct {
  Game_id string
  Rank uint
  Team_id string
  Team_player uint
  Created_at time.Time
  Updated_at time.Time
  Commands []byte
}

func (m *Model) CreateGame(ownerId string, firstBlock string) (string, error) {
  var err error
  gameKey, err := generateKey()
  if err != nil { return "", errors.Wrap(err, 0) }
  _, err = m.db.Exec(
    `INSERT INTO games (game_key, owner_id, first_block, last_block, current_round)
     VALUES (?, ?, ?, ?, 0)`, gameKey, ownerId, firstBlock, firstBlock)
  if err != nil { return "", errors.Wrap(err, 0) }
  return gameKey, nil
}

func (m *Model) ViewGame(gameKey string) (j.Value, error) {
  game, err := m.LoadGame(gameKey, NullFacet)
  if err != nil { return j.Null, err }
  if game == nil {
    return j.Null, nil
  }
  return viewGame(game), nil
}

func (m *Model) SetPlayerCommands(gameKey string, teamKey string, currentBlock string, teamPlayer uint, commands []byte) (err error) {
  teamId, err := m.FindTeamIdByKey(teamKey)
  if err != nil { return }
  err = m.transaction(func () error {
    game, err := m.LoadGame(gameKey, NullFacet)
    if err != nil { return err }
    if game.Last_block != currentBlock {
      return errors.New("current block has changed")
    }
    rank, err := m.getPlayerRank(game.Id, teamId, teamPlayer)
    if err != nil { return err }
    if rank == 0 {
      return m.addPlayerToGame(game.Id, teamId, teamPlayer, commands)
    } else {
      return m.setPlayerCommands(game.Id, rank, commands)
    }
  })
  return err
}

/*
func (m *Model) getGameId(gameKey string) (string, error) {
  row := m.db.QueryRow(`SELECT id FROM games WHERE game_key = ?`, gameKey)
  var id string
  err := row.Scan(&id)
  if err != nil { return "", err }
  return id, nil
}
*/

func (m *Model) getPlayerRank(gameId string, teamId string, teamPlayer uint) (uint, error) {
  row := m.db.QueryRow(
    `SELECT rank FROM game_players
      WHERE game_id = ? AND team_id = ? AND team_player = ? LIMIT 1`,
      gameId, teamId, teamPlayer)
  var rank uint
  err := row.Scan(&rank)
  if err == sql.ErrNoRows { return 0, nil }
  if err != nil { return 0, err }
  return rank, nil
}

func (m *Model) addPlayerToGame (gameId string, teamId string, teamPlayer uint, commands []byte) error {
  var err error
  _, err = m.db.Exec(
    `INSERT INTO game_players (game_id, rank, team_id, team_player, commands)
      SELECT ?, 1 + COUNT(rank), ?, ?, ?
      FROM game_players
      WHERE game_id = ?`,
    gameId, teamId, teamPlayer, commands, gameId)
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

func (m *Model) setPlayerCommands (gameId string, rank uint, commands []byte) error {
  var err error
  _, err = m.db.Exec(
    `UPDATE game_players
      SET commands = ?
      WHERE game_id = ? AND rank = ?`,
    commands, gameId, rank)
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

func (m *Model) loadPlayersOfGameTeam (gameKey string, teamId string, f Facets) ([]GamePlayer, error) {
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

func (m *Model) LoadGame(key string, f Facets) (*Game, error) {
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
    view.Prop("rank", j.Uint(res.Rank))
    view.Prop("teamPlayer", j.Uint(res.Team_player))
    timeProp(view, "createdAt", res.Created_at)
    timeProp(view, "updatedAt", res.Updated_at)
    view.Prop("commands", j.Raw(res.Commands))
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
  view.Prop("ownerId", j.String(game.Owner_id))
  view.Prop("firstBlock", j.String(game.First_block))
  view.Prop("lastBlock", j.String(game.Last_block))
  nullTimeProp(view, "startedAt", game.Started_at)
  nullTimeProp(view, "roundEndsAt", game.Round_ends_at)
  view.Prop("currentRound", j.Uint(game.Current_round))
  return view
}
