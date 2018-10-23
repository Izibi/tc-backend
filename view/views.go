
package view

import (
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

func (v *View) ViewUser(userId int64) error {
  v.userId = userId
  user, err := v.model.LoadUser(userId)
  if err != nil { return err }
  if user != nil {
    v.addUser(user)
    v.Set("userId", j.String(ExportId(user.Id)))
  } else {
    v.Set("userId", j.Null)
  }
  return nil
}

func (v *View) ViewUserContests(userId int64) error {
  v.userId = userId
  contests, err := v.model.LoadUserContests(userId)
  if err != nil { return err }
  contestIds := j.Array()
  for i := range contests {
    contest := &contests[i]
    v.addContest(contest)
    v.tasks.Need(contest.Task_id)
  }
  err = v.tasks.Load(v.loadTasks)
  if err != nil { return err }
  v.Set("contestIds", contestIds)
  return nil
}

func (v *View) ViewUserContest(userId int64, contestId int64) error {
  v.userId = userId
  v.contestId = contestId

  /* verify user has access to contest */
  ok, err := v.model.CanUserAccessContest(userId, contestId)
  if err != nil { return err }
  if !ok { return errors.Errorf("access denied") }

  contest, err := v.model.LoadContest(v.contestId)
  if err != nil { return err }

  v.addContest(contest)
  v.tasks.Need(contest.Task_id)
  err = v.tasks.Load(v.loadTasks)
  if err != nil { return err }
  err = v.loadTaskResources(contest.Task_id)
  if err != nil { return err }
  err = v.loadContestTeams(contestId)
  if err != nil { return err }

  return nil
}

func (v *View) ViewUserContestTeam(userId int64, contestId int64) error {
  v.userId = userId
  v.contestId = contestId
  team, err := v.model.LoadUserContestTeam(userId, contestId)
  if err != nil { return err }
  if team == nil {
    v.Set("teamId", j.Null)
    return nil
  }
  v.teamId = team.Id
  v.Set("teamId", j.String(ExportId(team.Id)))
  err = v.loadTeams([]int64{team.Id})
  if err != nil { return err }
  err = v.loadTeamMembers([]int64{team.Id})
  if err != nil { return err }
  return nil
}


type ChainFilters struct {
  Status string
  TeamId string
  TitleSearch string
}

func (v *View) ViewChains(userId int64, contestId int64, filters ChainFilters) error {
  /* Rely on ViewUserContest to perform access checking. */
  /* Load all contest teams?  or do that as a separate request? */
/*
  TODO: add support for filters:
  - status
  - teamId | null
  - text (chain title)
*/
  chains, err := v.model.LoadContestChains(v.contestId /* filters */)
  chainIds := j.Array()
  for i := range chains {
    chain := &chains[i]
    chainIds.Item(j.String(ExportId(chain.Id)))
    v.addChain(chain)
    if chain.Owner_id.Valid {
      v.teams.Need(chain.Owner_id.Int64)
    }
  }
  err = v.teams.Load(v.loadTeams)
  if err != nil { return errors.Wrap(err, 0) }
  v.Set("chainIds", chainIds)
  return nil
}
