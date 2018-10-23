
package view

import (
  "fmt"
  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/model"
)

func (v *View) loadTeams(ids []int64) error {
  teams, err := v.model.LoadTeamsById(ids)
  if err != nil { return err }
  for i := range teams {
    team := &teams[i]
    v.addTeam(team)
  }
  return nil
}

func (v *View) loadContestTeams(contestId int64) error {
  teams, err := v.model.LoadContestTeams(contestId)
  if err != nil { return err }
  ids := j.Array()
  for i := range teams {
    team := &teams[i]
    ids.Item(j.String(ExportId(team.Id)))
    v.addTeam(team)
  }
  obj := j.Object()
  obj.Prop("teamIds", ids)
  v.Add(fmt.Sprintf("contests#teams %s", ExportId(contestId)), obj)
  return nil
}

func (v *View) loadTasks(ids []int64) error {
  tasks, err := v.model.LoadTasksById(ids)
  if err != nil { return err }
  for i := range tasks {
    v.addTask(&tasks[i])
  }
  return nil
}

func (v *View) loadTeamMembers(ids []int64) error {
  allMembers, err := v.model.LoadTeamMembersByTeamId(ids)
  if err != nil { return err }
  byTeamId := make(map[int64][]*model.TeamMember)
  for i := range allMembers {
    m := &allMembers[i]
    byTeamId[m.Team_id] = append(byTeamId[m.Team_id], m)
  }
  for teamId, members := range byTeamId {
    ids := j.Array()
    for _, member := range members {
      ids.Item(j.String(v.addTeamMember(member)))
    }
    obj := j.Object()
    obj.Prop("memberIds", ids)
    v.Add(fmt.Sprintf("teams#members %s", ExportId(teamId)), obj)
  }
  return nil
}

func (v *View) loadTaskResources(taskId int64) error {
  taskResources, err := v.model.LoadTaskResources(taskId)
  if err != nil { return err }
  ids := j.Array()
  for i := range taskResources {
    res := &taskResources[i]
    ids.Item(j.String(ExportId(res.Id)))
    v.addTaskResource(res)
  }
  obj := j.Object()
  obj.Prop("resourceIds", ids)
  v.Add(fmt.Sprintf("tasks#resources %s", ExportId(taskId)), obj)
  return nil
}
