
package model

import (
  "database/sql"
  "fmt"
  "time"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

type Chain struct {
  Id int64
  Created_at string
  Updated_at string
  Contest_id int64
  Owner_id sql.NullInt64
  Parent_id sql.NullString
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
  Round int
}

type ChainFilters struct {
  Status string
  teamId string
  titleSearch string
}

func (m *Model) ViewChains(userId int64, contestId int64, filters ChainFilters) error {
  /* Rely on ViewUserContest to perform access checking. */
  /* Load all contest teams?  or do that as a separate request? */
  err := m.ViewUserContest(userId, contestId)
  if err != nil { return nil }
/*
  TODO: add filters:
  - status
  - teamId | null
  - text (chain title)
*/
  rows, err := m.db.Queryx(`SELECT * FROM chains WHERE contest_id = ?`, contestId)
  if err != nil { return errors.Wrap(err, 0) }
  defer rows.Close()
  chainIds := j.Array()
  for rows.Next() {
    chain, err := m.loadChainRow(rows, BaseFacet)
    if err != nil { return err }
    chainIds.Item(j.String(m.ExportId(chain.Id)))
    if chain.Owner_id.Valid {
      m.teams.Need(chain.Owner_id.Int64)
    }
  }
  m.teams.Load(m.loadTeams)
  m.Set("chainIds", chainIds)
  return nil
}

func (m *Model) ForkChain(teamId int64, chainId int64) (int64, error) {
  var err error
  var chain Chain
  err = m.dbMap.Get(&chain, chainId)
  if err != nil { return 0, err }
  now := time.Now().Format(time.RFC3339)
  newChain := &Chain{
    Created_at: now,
    Updated_at: now,
    Contest_id: chain.Contest_id,
    Owner_id: sql.NullInt64{teamId, true},
    Parent_id: sql.NullString{m.ExportId(chain.Id), true},
    Status_id: 1 /* private test */,
    Title: fmt.Sprintf("forked from %s", chain.Title),
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
    Round: 0,
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
  team, err := m.LoadUserContestTeam(userId, chain.Contest_id, BaseFacet)
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

func (m *Model) LoadChain(chainId int64, f Facets) (*Chain, error) {
  return m.loadChainRow(m.db.QueryRowx(
    `SELECT * FROM chains WHERE id = ?`, chainId), f)
}

func (m *Model) loadChainRow(row IRow, f Facets) (*Chain, error) {
  var res Chain
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  if f.Base {
    view := j.Object()
    view.Prop("id", j.String(m.ExportId(res.Id)))
    view.Prop("createdAt", j.String(res.Created_at))
    view.Prop("updatedAt", j.String(res.Updated_at))
    view.Prop("contestId", j.String(m.ExportId(res.Contest_id)))
    ownerId := j.Null
    if res.Owner_id.Valid {
      ownerId = j.String(m.ExportId(res.Owner_id.Int64))
    }
    view.Prop("ownerId", ownerId)
    parentId := j.Null
    if res.Parent_id.Valid {
      parentId = j.String(res.Parent_id.String)
    }
    view.Prop("parentId", parentId)
    view.Prop("statusId", j.String(m.ExportId(res.Status_id))) // XXX
    view.Prop("title", j.String(res.Title))
    view.Prop("description", j.String(res.Description))
    view.Prop("interfaceText", j.String(res.Interface_text))
    view.Prop("implementationText", j.String(res.Implementation_text))
    view.Prop("protocolHash", j.String(res.Protocol_hash))
    view.Prop("newProtocolHash", j.String(res.New_protocol_hash))
    startedAt := j.Null
    if res.Started_at.Valid {
      startedAt = j.String(res.Started_at.String)
    }
    view.Prop("startedAt", startedAt)
    view.Prop("currentGameKey", j.String(res.Game_key))
    view.Prop("currentRound", j.Int(res.Round))
    view.Prop("nbVotesApprove", j.Int(res.Nb_votes_approve))
    view.Prop("nbVotesReject", j.Int(res.Nb_votes_reject))
    view.Prop("nbVotesUnknown", j.Int(res.Nb_votes_unknown))
    m.Add(fmt.Sprintf("chains %s", m.ExportId(res.Id)), view)
  }
  return &res, nil
}
