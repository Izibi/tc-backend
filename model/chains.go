
package model

import (
  "database/sql"
  "time"
  "github.com/go-errors/errors"
)

type Chain struct {
  Id int64
  Created_at time.Time
  Updated_at time.Time
  Contest_id int64
  Owner_id sql.NullInt64
  Parent_id sql.NullInt64
  Status_id int64
  Title string
  Description string `db:"description_text"`
  Interface_text string
  Implementation_text string
  Protocol_hash string
  New_protocol_hash string
  Started_at sql.NullString
  Game_key string
  Nb_votes_reject int
  Nb_votes_unknown int
  Nb_votes_approve int
}

type ChainStatusFilter struct {
  Status string
  TeamId int64
}

func (m *Model) LoadChain(chainId int64) (*Chain, error) {
  var err error
  var chain Chain
  err = m.dbMap.Get(&chain, chainId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return &chain, nil
}

func (m *Model) LoadContestChains(contestId int64, filters... interface{}) ([]Chain, error) {
  var chains []Chain
  query := `SELECT id,created_at,updated_at,started_at,status_id,owner_id,title,game_key,parent_id,protocol_hash,nb_votes_approve,nb_votes_reject,nb_votes_unknown,contest_id FROM chains WHERE contest_id = ?`
  args := []interface{}{contestId}
  for _, f := range filters {
    switch filter := f.(type) {
    case ChainStatusFilter:
      switch filter.Status {
      case "main":
        query = query + ` AND status_id = 4`
      case "private_test":
        query = query + ` AND status_id = 1 AND owner_id = ?`
        args = append(args, filter.TeamId)
      case "public_test":
        query = query + ` AND status_id = 2`
      case "candidate":
        query = query + ` AND status_id = 3`
      case "past":
        query = query + ` AND status_id = 5`
      }
    }
  }
  // fmt.Printf("query %s %v\n", query, args)
  err := m.dbMap.Select(&chains, query, args...)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return chains, nil
}

func (m *Model) ForkChain(teamId int64, chainId int64, title string) (int64, error) {
  var err error
  var chain Chain
  err = m.dbMap.Get(&chain, chainId)
  if err != nil { return 0, err }
  now := time.Now()
  newChain := &Chain{
    Created_at: now,
    Updated_at: now,
    Contest_id: chain.Contest_id,
    Owner_id: sql.NullInt64{teamId, true},
    Parent_id: sql.NullInt64{chain.Id, true},
    Status_id: 1 /* private test */,
    Title: title,
    Description: chain.Description,
    Interface_text: chain.Interface_text,
    Implementation_text: chain.Implementation_text,
    Protocol_hash: chain.Protocol_hash,
    New_protocol_hash: chain.New_protocol_hash,
    Started_at: sql.NullString{},
    Game_key: "",
    Nb_votes_reject: 0,
    Nb_votes_unknown: 0,
    Nb_votes_approve: 0,
  }
  err = m.dbMap.Insert(newChain)
  if err != nil { return 0, errors.Wrap(err, 0) }
  if newChain.Id == 0 {
    return 0, errors.New("insert failed")
  }
  /* TODO: post to the team's channel, an event indicating that a new chain
     has been created */
  return newChain.Id, nil
}

func (m *Model) DeleteChain(userId int64, chainId int64) (*Chain, error) {
  /*
    A private chain can be deleted by its team members.
  */
  var err error
  var chain Chain
  err = m.dbMap.Get(&chain, chainId)
  if err != nil { return nil, err }
  team, err := m.LoadUserContestTeam(userId, chain.Contest_id)
  if err != nil { return nil, err }
  if team == nil || team.Id != chain.Owner_id.Int64 {
    return nil, errors.New("access denied")
  }
  if chain.Status_id != 1 { // FIXME hard-coded id in select id from chain_statuses where is_public = 0
    return nil, errors.New("forbidden")
  }
  _, err = m.db.Exec(`DELETE FROM chains WHERE id = ?`, chainId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return &chain, nil
}

func (m *Model) SetChainGameKey(chainId int64, gameKey string) error {
  var err error
  _, err = m.db.Exec(
    `UPDATE chains SET game_key = ? WHERE id = ?`, gameKey, chainId)
  if err != nil { return err }
  return nil
}

func (m *Model) SaveChain(chain *Chain) error {
  _, err := m.dbMap.Update(chain)
  return err
}
