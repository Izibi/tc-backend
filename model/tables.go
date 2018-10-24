package model

import (
  "github.com/jmoiron/modl"
)

type Tables struct {
  chains *modl.TableMap
  chainRevisions *modl.TableMap
  contests *modl.TableMap
  games *modl.TableMap
  gamePlayers *modl.TableMap
  tasks *modl.TableMap
  taskResources *modl.TableMap
  teamMembers *modl.TableMap
  teams *modl.TableMap
  users *modl.TableMap

  //chainStatuses *modl.TableMap
}

func (t *Tables) Map(m *modl.DbMap) {
  t.chains = m.AddTableWithName(Chain{}, "chains").SetKeys(true, "Id")
  t.chainRevisions = m.AddTableWithName(ChainRevision{}, "chain_revisions").SetKeys(true, "Id")
  t.contests = m.AddTableWithName(Contest{}, "contests").SetKeys(true, "Id")
  t.games = m.AddTableWithName(Game{}, "games").SetKeys(true, "Id")
  t.gamePlayers = m.AddTableWithName(GamePlayer{}, "game_players").SetKeys(true, "Game_id", "Rank")
  t.users = m.AddTableWithName(User{}, "users").SetKeys(true, "Id")
  t.taskResources = m.AddTableWithName(TaskResource{}, "task_resources").SetKeys(true, "Id")
  t.tasks = m.AddTableWithName(Task{}, "tasks").SetKeys(true, "Id")
  t.teamMembers = m.AddTableWithName(TeamMember{}, "team_members").SetKeys(true, "Team_id", "User_id")
  t.teams = m.AddTableWithName(Team{}, "teams").SetKeys(true, "Id")
  t.users = m.AddTableWithName(User{}, "users").SetKeys(true, "Id")
  //t.chainStatuses = m.AddTableWithName(ChainStatus{}, "chain_statuses").SetKeys(true, "Id")
}
