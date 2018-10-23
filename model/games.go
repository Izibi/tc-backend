
package model

import (
  "database/sql"
  "encoding/json"
  "fmt"
  "time"
  "github.com/go-sql-driver/mysql"
  "github.com/go-errors/errors"
  ji "github.com/json-iterator/go"
  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/utils"
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
  Rank uint32
  Team_id int64
  Team_player uint32 // TODO: rename Bot_id
  Created_at time.Time
  Updated_at time.Time
  Locked_at *time.Time
  Commands []byte
  Used []byte
  Unused []byte
}

type RegisteredGamePlayer struct {
  Rank uint32
  Team_id int64
  Team_player uint32 // TODO: rename Bot_id
}

type PlayerInput struct {
  Rank uint32
  Commands []json.RawMessage
  Used []byte
  Unused []byte
}

func (m *Model) LoadGame(key string) (*Game, error) {
  var game Game
  err := m.dbMap.SelectOne(&game,
    `SELECT * FROM games WHERE game_key = ?`, key)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  return &game, nil
}

func (m *Model) loadGameForUpdate(key string) (*Game, error) {
  var game Game
  err := m.dbMap.SelectOne(&game,
    `SELECT * FROM games WHERE game_key = ? FOR UPDATE`, key)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  return &game, nil
}

func (m *Model) CreateGame(ownerId int64, firstBlock string, currentRound uint64) (string, error) {
  var err error
  gameKey, err := utils.NewKey()
  var nbCyclesPerRound = 2
  if err != nil { return "", errors.Wrap(err, 0) }
  _, err = m.db.Exec(
    `INSERT INTO games (game_key, owner_id, first_block, last_block, current_round, nb_cycles_per_round, next_block_commands)
     VALUES (?, ?, ?, ?, ?, ?, "")`, gameKey, ownerId, firstBlock, firstBlock, currentRound, nbCyclesPerRound)
  if err != nil { return "", errors.Wrap(err, 0) }
  return gameKey, nil
}

func (m *Model) RegisterGamePlayers(gameKey string, teamId int64, botIds []uint32) (ranks []uint32, err error) {
  err = m.transaction(func () error {
    var err error
    var game *Game
    game, err = m.LoadGame(gameKey)
    if err != nil { return err }
    if game == nil { return errors.New("bad game key") }
    var ps []RegisteredGamePlayer
    ps, err = m.loadRegisteredGamePlayer(game.Id)
    var nextRank uint32 = uint32(len(ps)) + 1
    bot_loop: for _, botId := range botIds {
      for _, p := range ps {
        if p.Team_id == teamId && botId == p.Team_player {
          ranks = append(ranks, p.Rank)
          continue bot_loop
        }
      }
      p := RegisteredGamePlayer{
        Rank: nextRank,
        Team_id: teamId,
        Team_player: botId,
      }
      err = m.addPlayerToGame(game.Id, &p)
      if err != nil { return err }
      nextRank += 1
      ranks = append(ranks, p.Rank)
    }
    return nil
  })
  if err != nil { return nil, err }
  return ranks, nil
}

func (m *Model) SetPlayerCommands(gameKey string, currentBlock string, teamId int64, teamPlayer uint32, commands []byte) (err error) {
  err = m.transaction(func () error {
    game, err := m.LoadGame(gameKey)
    if err != nil { return err }
    if game.Last_block != currentBlock {
      return errors.New("current block has changed")
    }
    fmt.Printf("setPlayerCommands %d %d %d\n", game.Id, teamId, teamPlayer)
    return m.setPlayerCommands(game.Id, teamId, teamPlayer, commands)
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
    game, err = m.loadGameForUpdate(gameKey)
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
    // game, err = m.LoadGame(gameKey)
    game.Next_block_commands = commands
    game.Locked = true
    return nil
  })
  if err !=  nil { return nil, err }
  return game, nil
}

func (m *Model) CancelRound(gameKey string) error {
  return m.transaction(func () error {
    game, err := m.loadGameForUpdate(gameKey)
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
    game, err := m.loadGameForUpdate(gameKey)
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

func (m *Model) loadRegisteredGamePlayer(gameId int64) ([]RegisteredGamePlayer, error) {
  var err error
  rows, err := m.db.Queryx(
    `SELECT rank, team_id, team_player FROM game_players WHERE game_id = ? ORDER by rank`, gameId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  defer rows.Close()
  var ps []RegisteredGamePlayer
  for rows.Next() {
    var p RegisteredGamePlayer
    err = rows.StructScan(&p)
    if err != nil { return nil, err }
    ps = append(ps, p)
  }
  return ps, nil
}

func (m *Model) getPlayerRank(gameId int64, teamId int64, teamPlayer uint32) (uint32, error) {
  row := m.db.QueryRow(
    `SELECT rank FROM game_players
      WHERE game_id = ? AND team_id = ? AND team_player = ? LIMIT 1`,
      gameId, teamId, teamPlayer)
  var rank uint32
  err := row.Scan(&rank)
  if err == sql.ErrNoRows { return 0, nil }
  if err != nil { return 0, err }
  return rank, nil
}

func (m *Model) addPlayerToGame(gameId int64, player *RegisteredGamePlayer) error {
  var err error
  _, err = m.db.Exec(
    `INSERT INTO game_players (game_id, rank, team_id, team_player, commands, used, unused)
      VALUES (?, ?, ?, ?, "", "", "")`, /* team_player -> bot_id */
    gameId, player.Rank, player.Team_id, player.Team_player)
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

func (m *Model) setPlayerCommands(gameId int64, teamId int64, teamPlayer uint32, commands []byte) error {
  var err error
  _, err = m.db.Exec(
    `UPDATE game_players
      SET commands = ?
      WHERE game_id = ? AND team_id = ? AND team_player = ?`,
    commands, gameId, teamId, teamPlayer)
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

func (m *Model) LoadGamePlayerTeamKeys(gameKey string) ([]string, error) {
  var err error
  rows, err := m.db.Queryx(
    `SELECT DISTINCT t.public_key FROM teams t
      INNER JOIN game_players gp ON gp.team_id = t.id
      INNER JOIN games g ON g.id = gp.game_id
      WHERE g.game_key = ?
      ORDER BY t.public_key`, gameKey)
  if err != nil { return nil, errors.Wrap(err, 0) }
  defer rows.Close()
  var items []string
  for rows.Next() {
    var key string
    err = rows.Scan(&key)
    if err != nil { return nil, err }
    items = append(items, key)
  }
  return items, nil
}

func (m *Model) getNextBlockCommands (gameId int64, nbCycles uint) ([]byte, error) {
  var err error
  rows, err := m.db.Queryx(
    `SELECT rank, commands FROM game_players gp
     WHERE game_id = ? ORDER BY rank FOR UPDATE`, gameId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  defer rows.Close()
  var commands = j.Array()
  var cycles = make([]j.IArray, nbCycles, nbCycles)
  var i uint
  for i = 0; i < nbCycles; i++ {
    cycleCmds := j.Array()
    commands.Item(cycleCmds)
    cycles[i] = cycleCmds
  }
  for rows.Next() {
    var player GamePlayer
    err := rows.Scan(&player)
    if err != nil { return nil, errors.Wrap(err, 0) }
    input, err := preparePlayerInput(player.Rank, player.Commands, nbCycles)
    if err != nil { return nil, err }
    _, err = m.db.Exec(
      `UPDATE game_players
       SET used = ?, unused = ?, updated_at = NOW()
       WHERE game_id = ? AND rank = ?`,
       input.Used, input.Unused, gameId, player.Rank)
    if err != nil { return nil, errors.Wrap(err, 0) }
    for i, cmd := range input.Commands {
      obj := j.Object()
      obj.Prop("player", j.Uint32(player.Rank))
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

func preparePlayerInput(rank uint32, commands []byte, nbCycles uint) (*PlayerInput, error) {
  var cmds []json.RawMessage
  err := json.Unmarshal([]byte(commands), &cmds)
  if err != nil { return nil, err }
  nbCommands := len(cmds)
  firstUnused := int(nbCycles)
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
