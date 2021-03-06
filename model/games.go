
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
  Current_round uint64
  Max_nb_rounds uint64
  Max_nb_players uint32
  Nb_cycles_per_round uint32
}

type GameParams struct {
  First_round uint64 `json:"first_round"` // Current_round
  Nb_rounds uint64 `json:"nb_rounds"` // Max_nb_rounds
  Nb_players uint32 `json:"nb_players"` // Max_nb_players
  Cycles_per_round uint32 `json:"cycles_per_round"` // Nb_cycles_per_round
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

type GamePlayerId struct {
  Rank uint32
  Team_key string
  Bot_id uint32
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

func (m *Model) IsGameOwner(gameKey string, teamId int64) (bool, error) {
  row := m.db.QueryRow(`SELECT 1 FROM games WHERE game_key = ? AND owner_id = ?`, gameKey, teamId)
  var ok bool
  err := row.Scan(&ok)
  if err == sql.ErrNoRows { return false, nil }
  if err != nil { return false, err }
  return true, nil
}

func (m *Model) CreateGame(ownerId int64, firstBlock string, params GameParams) (string, error) {
  var err error
  gameKey, err := utils.NewKey()
  if err != nil { return "", errors.Wrap(err, 0) }
  now := time.Now()
  game := Game{
    Game_key: gameKey,
    Created_at: now,
    Updated_at: now,
    Owner_id: ownerId,
    First_block: firstBlock,
    Last_block: firstBlock,
    Started_at: mysql.NullTime{},
    Round_ends_at: mysql.NullTime{},
    Locked: false,
    Next_block_commands: []byte{},
    Current_round: params.First_round,
    Max_nb_rounds: params.Nb_rounds,
    Max_nb_players: params.Nb_players,
    Nb_cycles_per_round: params.Cycles_per_round,
  }
  err = m.dbMap.Insert(&game)
  if err != nil { return "", errors.Wrap(err, 0) }
  return gameKey, nil
}

func (m *Model) RegisterGamePlayers(gameKey string, teamId int64, botIds []uint32) ([]uint32, error) {
  var err error
  var game *Game
  game, err = m.LoadGame(gameKey)
  if err != nil { return nil, err }
  if game == nil { return nil, errors.New("bad game key") }
  var ps []RegisteredGamePlayer
  ps, err = m.LoadRegisteredGamePlayer(game.Id)
  var ranks []uint32
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
    if len(ps) < int(game.Max_nb_players) {
      err = m.addPlayerToGame(game.Id, &p)
      if err != nil { return nil, err }
      nextRank += 1
      ps = append(ps, p)
      ranks = append(ranks, p.Rank)
    }
  }
  return ranks, nil
}

func (m *Model) SetPlayerCommands(gameKey string, currentBlock string, teamId int64, teamPlayer uint32, commands []byte) (err error) {
  game, err := m.LoadGame(gameKey)
  if err != nil { return err }
  if game.Last_block != currentBlock {
    return errors.New("current block has changed")
  }
  fmt.Printf("setPlayerCommands %d %d %d\n", game.Id, teamId, teamPlayer)
  return m.setPlayerCommands(game.Id, teamId, teamPlayer, commands)
}

func (m *Model) CloseRound(gameKey string, currentBlock string) (*Game, error) {
  var err error
  var commands []byte
  var game *Game
  game, err = m.loadGameForUpdate(gameKey)
  if err != nil { return nil, err }
  if game.Current_round >= game.Max_nb_rounds {
    return game, errors.New("game has ended")
  }
  if game.Last_block != currentBlock {
    return game, errors.New("current block has changed")
  }
  if game.Locked {
    return game, errors.New("game is locked")
  }
  commands, err = m.getNextBlockCommands(game.Id, game.Nb_cycles_per_round)
  if err != nil { return game, err }
  game.Next_block_commands = commands
  err = m.lockGame(game.Id, commands)
  if err != nil { return game, err }
  game.Locked = true
  return game, nil
}

func (m *Model) CancelRound(gameKey string) (*Game, error) {
  game, err := m.loadGameForUpdate(gameKey)
  if err != nil { return nil, err }
  _, err = m.db.Exec(
    `UPDATE game_players SET locked_at = NULL WHERE game_id = ?`, game.Id)
  if err != nil { return game, err }
  _, err = m.db.Exec(
    `UPDATE games SET locked = 0, updated_at = NOW() WHERE id = ?`, game.Id)
  if err != nil { return game, errors.Wrap(err, 0) }
  game.Locked = false
  return game, nil
}

func (m *Model) EndRoundAndUnlock(gameKey string, newBlock string) (*Game, error) {
  game, err := m.loadGameForUpdate(gameKey)
  if err != nil { return nil, err }
  if !game.Locked { return game, errors.New("game is not locked") }
  _, err = m.db.Exec(
    `UPDATE game_players SET
      locked_at = NULL,
      commands = IF(updated_at > locked_at, commands, unused)
     WHERE game_id = ?`, game.Id)
  if err != nil { return game, err }
  _, err = m.db.Exec(
    `UPDATE games SET
      locked = 0,
      current_round = current_round + 1,
      last_block = ?,
      next_block_commands = "",
      updated_at = NOW()
     WHERE id = ?`, newBlock, game.Id)
  if err != nil { return game, errors.Wrap(err, 0) }
  game.Locked = false
  game.Current_round += 1
  game.Last_block = newBlock
  game.Next_block_commands = []byte{}
  return game, nil
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

func (m *Model) LoadRegisteredGamePlayer(gameId int64) ([]RegisteredGamePlayer, error) {
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

func (m *Model) LoadGamePlayerIds(gameKey string) ([]GamePlayerId, error) {
  var err error
  rows, err := m.db.Queryx(
    `SELECT gp.rank, t.public_key AS team_key, gp.team_player AS bot_id FROM teams t
      INNER JOIN game_players gp ON gp.team_id = t.id
      INNER JOIN games g ON g.id = gp.game_id
      WHERE g.game_key = ?
      ORDER BY gp.rank`, gameKey)
  if err != nil { return nil, errors.Wrap(err, 0) }
  defer rows.Close()
  var items []GamePlayerId
  for rows.Next() {
    var item GamePlayerId
    err = rows.StructScan(&item)
    if err != nil { return nil, err }
    items = append(items, item)
  }
  return items, nil
}

func (m *Model) getNextBlockCommands (gameId int64, nbCycles uint32) ([]byte, error) {
  var err error
  rows, err := m.db.Queryx(
    `SELECT rank, commands FROM game_players gp
     WHERE game_id = ? ORDER BY rank FOR UPDATE`, gameId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  defer rows.Close()
  var commands = j.Array()
  var cycles = make([]j.IArray, nbCycles, nbCycles)
  var i uint32
  for i = 0; i < nbCycles; i++ {
    cycleCmds := j.Array()
    commands.Item(cycleCmds)
    cycles[i] = cycleCmds
  }
  for rows.Next() {
    var player GamePlayer
    err := rows.Scan(&player.Rank, &player.Commands)
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

func preparePlayerInput(rank uint32, commands []byte, nbCycles uint32) (*PlayerInput, error) {
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
