
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
  Id int64
  Game_key string
  Created_at time.Time
  Updated_at time.Time
  Owner_id int64
  First_block string
  Last_block string
  Started_at mysql.NullTime
  Round_ends_at mysql.NullTime
  Locked bool
  Next_block_commands []byte
  Nb_cycles_per_round uint
  Current_round uint64
}

type GamePlayer struct {
  Game_id int64
  Rank uint
  Team_id int64
  Team_player uint
  Created_at time.Time
  Updated_at time.Time
  Locked_at *time.Time
  Commands []byte
  Used []byte
  Unused []byte
}

type PlayerInput struct {
  Rank uint
  Commands []json.RawMessage
  Used []byte
  Unused []byte
}

func (m *Model) CreateGame(ownerId int64, firstBlock string, currentRound uint64) (string, error) {
  var err error
  gameKey, err := generateKey()
  var nbCyclesPerRound = 2
  if err != nil { return "", errors.Wrap(err, 0) }
  _, err = m.db.Exec(
    `INSERT INTO games (game_key, owner_id, first_block, last_block, current_round, nb_cycles_per_round, next_block_commands)
     VALUES (?, ?, ?, ?, ?, ?, "")`, gameKey, ownerId, firstBlock, firstBlock, currentRound, nbCyclesPerRound)
  if err != nil { return "", errors.Wrap(err, 0) }
  return gameKey, nil
}

func (m *Model) SetPlayerCommands(gameKey string, teamKey string, currentBlock string, teamPlayer uint, commands []byte) (err error) {
  teamId, err := m.FindTeamIdByKey(teamKey)
  if err != nil { return errors.New("team key is not recognized")}
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

func (m *Model) CloseRound(gameKey string, teamKey string, currentBlock string) (*Game, error) {
  teamId, err := m.FindTeamIdByKey(teamKey)
  if err != nil { return nil, err }
  var commands []byte
  var game *Game
  err = m.transaction(func () error {
    var err error
    game, err = m.loadGameForUpdate(gameKey, NullFacet)
    if err != nil { return err }
    if game.Last_block != currentBlock {
      return errors.New("current block has changed")
    }
    if game.Owner_id != teamId {
      return errors.New("only the game owner can end a round")
    }
    if game.Locked {
      return errors.New("game is locked")
    }
    commands, err = m.getNextBlockCommands(game.Id, game.Nb_cycles_per_round)
    if err != nil { return err }
    err = m.lockGame(game.Id, commands)
    if err != nil { return err }
    // game, err = m.LoadGame(gameKey, NullFacet)
    game.Next_block_commands = commands
    game.Locked = true
    return nil
  })
  if err !=  nil { return nil, err }
  return game, nil
}

func (m *Model) CancelRound(gameKey string) error {
  return m.transaction(func () error {
    game, err := m.loadGameForUpdate(gameKey, NullFacet)
    if err != nil { return err }
    _, err = m.db.Exec(
      `UPDATE game_players SET locked_at = NULL WHERE game_id = ?`, game.Id)
    if err != nil { return err }
    _, err = m.db.Exec(
      `UPDATE games SET locked = 0 WHERE id = ?`, game.Id)
    if err != nil { return errors.Wrap(err, 0) }
    return nil
  })
}

func (m *Model) EndRoundAndUnlock(gameKey string, newBlock string) error {
  return m.transaction(func () error {
    game, err := m.loadGameForUpdate(gameKey, NullFacet)
    if err != nil { return err }
    if !game.Locked { return errors.New("game is not locked") }
    _, err = m.db.Exec(
      `UPDATE game_players SET
        locked_at = NULL,
        commands = IF(updated_at > locked_at, commands, unused)
       WHERE game_id = ?`, game.Id)
    if err != nil { return err }
    _, err = m.db.Exec(
      `UPDATE games SET
        locked = 0,
        current_round = current_round + 1,
        last_block = ?,
        next_block_commands = ""
       WHERE id = ?`, newBlock, game.Id)
    if err != nil { return errors.Wrap(err, 0) }
    return nil
  })
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

func (m *Model) getPlayerRank(gameId int64, teamId int64, teamPlayer uint) (uint, error) {
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

func (m *Model) addPlayerToGame (gameId int64, teamId int64, teamPlayer uint, commands []byte) error {
  var err error
  _, err = m.db.Exec(
    `INSERT INTO game_players (game_id, rank, team_id, team_player, commands, used, unused)
      SELECT ?, 1 + COUNT(rank), ?, ?, ?, "", ""
      FROM game_players
      WHERE game_id = ?`,
    gameId, teamId, teamPlayer, commands, gameId)
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

func (m *Model) setPlayerCommands (gameId int64, rank uint, commands []byte) error {
  var err error
  _, err = m.db.Exec(
    `UPDATE game_players
      SET commands = ?
      WHERE game_id = ? AND rank = ?`,
    commands, gameId, rank)
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

func (m *Model) loadPlayersOfGameTeam (gameKey string, teamId int64, f Facets) ([]GamePlayer, error) {
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

func (m *Model) getNextBlockCommands (gameId int64, count uint) ([]byte, error) {
  var err error
  rows, err := m.db.Queryx(
    `SELECT rank, commands FROM game_players gp WHERE game_id = ? ORDER BY rank`, gameId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  defer rows.Close()
  var commands = j.Array()
  var cycles = make([]j.IArray, count, count)
  var i uint
  for i = 0; i < count; i++ {
    cycleCmds := j.Array()
    commands.Item(cycleCmds)
    cycles[i] = cycleCmds
  }
  for rows.Next() {
    item, err := m.loadGamePlayerRow(rows, NullFacet)
    if err != nil { return nil, err }
    input, err := preparePlayerInput(item.Rank, item.Commands, count)
    if err != nil { return nil, err }
    _, err = m.db.Exec(
      `UPDATE game_players SET used = ?, unused = ? WHERE game_id = ? AND rank = ?`,
        input.Used, input.Unused, gameId, item.Rank)
    if err != nil { return nil, errors.Wrap(err, 0) }
    for i, cmd := range input.Commands {
      obj := j.Object()
      obj.Prop("player", j.Uint(item.Rank))
      obj.Prop("command", j.String(ji.Get(cmd, "text").ToString()))
      cycles[i].Item(obj)
    }
  }
  _, err = m.db.Exec(
    `UPDATE game_players SET locked_at = NOW() WHERE game_id = ?`, gameId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  res, err := j.ToBytes(commands)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return res, nil
}

func preparePlayerInput(rank uint, commands []byte, count uint) (*PlayerInput, error) {
  var cmds []json.RawMessage
  err := json.Unmarshal([]byte(commands), &cmds)
  if err != nil { return nil, err }
  nbCommands := len(cmds)
  firstUnused := int(count)
  if firstUnused > nbCommands {
    firstUnused = nbCommands
  }
  usedCommands := cmds[0:firstUnused]
  used, err := json.Marshal(usedCommands)
  if err != nil { return nil, err }
  unused, err := json.Marshal(cmds[firstUnused:nbCommands])
  if err != nil { return nil, err }
  return &PlayerInput{
    Rank: rank,
    Commands: usedCommands,
    Used: used,
    Unused: unused,
  }, nil
}

func (m *Model) lockGame (gameId int64, commands []byte) error {
  res, err := m.db.Exec(
    `UPDATE games SET locked = 1, next_block_commands = ? WHERE id = ? AND locked = 0`,
      commands, gameId)
  if err != nil { return errors.Wrap(err, 0) }
  if n, err := res.RowsAffected(); err != nil || n == 0 {
    return errors.New("failed to lock game")
  }
  return nil
}

func (m *Model) LoadGame(key string, f Facets) (*Game, error) {
  return m.loadGameRow(m.db.QueryRowx(
    `SELECT * FROM games WHERE game_key = ?`, key), f)
}

func (m *Model) loadGameForUpdate(key string, f Facets) (*Game, error) {
  return m.loadGameRow(m.db.QueryRowx(
    `SELECT * FROM games WHERE game_key = ? FOR UPDATE`, key), f)
}

func (m *Model) loadGameRow(row IRow, f Facets) (*Game, error) {
  var res Game
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  if f.Base {
    m.Add(fmt.Sprintf("games %s", m.ExportId(res.Id)), m.ViewGame(&res))
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

func (m *Model) ViewGame(game *Game) j.Value {
  if game == nil {
    return j.Null
  }
  view := j.Object()
  view.Prop("key", j.String(game.Game_key))
  timeProp(view, "createdAt", game.Created_at)
  timeProp(view, "updatedAt", game.Updated_at)
  view.Prop("ownerId", j.String(m.ExportId(game.Owner_id)))
  view.Prop("firstBlock", j.String(game.First_block))
  view.Prop("lastBlock", j.String(game.Last_block))
  nullTimeProp(view, "startedAt", game.Started_at)
  nullTimeProp(view, "roundEndsAt", game.Round_ends_at)
  view.Prop("isLocked", j.Boolean(game.Locked))
  view.Prop("currentRound", j.Uint64(game.Current_round))
  view.Prop("nbCyclesPerRound", j.Uint(game.Nb_cycles_per_round))
  return view
}
