
package model

import (
  "database/sql"
  "fmt"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

type Chain struct {
  Id string
  Created_at string
  Updated_at string
  Contest_id string
  Owner_id sql.NullString
  Parent_id sql.NullString
  Status_id string
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

func (m *Model) ViewChains(userId string, contestId string, filters ChainFilters) error {
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
    chainIds.Item(j.String(chain.Id))
    if chain.Owner_id.Valid {
      m.teams.Need(chain.Owner_id.String)
    }
  }
  m.teams.Load(m.loadTeams)
  m.Set("chainIds", chainIds)
  return nil
}

func (m *Model) loadChainRow(row IRow, f Facets) (*Chain, error) {
  var res Chain
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  if f.Base {
    view := j.Object()
    view.Prop("id", j.String(res.Id))
    view.Prop("createdAt", j.String(res.Created_at))
    view.Prop("updatedAt", j.String(res.Updated_at))
    view.Prop("contestId", j.String(res.Contest_id))
    ownerId := j.Null
    if res.Owner_id.Valid {
      ownerId = j.String(res.Owner_id.String)
    }
    view.Prop("ownerId", ownerId)
    parentId := j.Null
    if res.Parent_id.Valid {
      parentId = j.String(res.Parent_id.String)
    }
    view.Prop("parentId", parentId)
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
    m.Add(fmt.Sprintf("chains %s", res.Id), view)
  }
  return &res, nil
}
