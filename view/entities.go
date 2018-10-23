
package view

import (
  "fmt"
  "tezos-contests.izibi.com/backend/model"
  j "tezos-contests.izibi.com/backend/jase"
)

func (v *View) addUser(user *model.User) string {
  id := ExportId(user.Id)
  obj := j.Object()
  obj.Prop("id", j.String(ExportId(user.Id)))
  obj.Prop("username", j.String(user.Username))
  obj.Prop("firstname", j.String(user.Firstname))
  obj.Prop("lastname", j.String(user.Lastname))
  if v.isAdmin {
    obj.Prop("foreignId", j.String(user.Foreign_id))
    obj.Prop("createdAt", j.String(user.Created_at))
    obj.Prop("updatedAt", j.String(user.Updated_at))
    obj.Prop("isAdmin", j.Boolean(user.Is_admin))
  }
  v.Add(fmt.Sprintf("users %s", id), obj)
  return id
}

func (v *View) addTeam(team *model.Team) string {
  id := ExportId(team.Id)
  obj := j.Object()
  obj.Prop("id", j.String(ExportId(team.Id)))
  obj.Prop("createdAt", j.String(team.Created_at))
  obj.Prop("updatedAt", j.Time(team.Updated_at))
  if team.Deleted_at.Valid {
    obj.Prop("deletedAt", j.String(team.Deleted_at.String))
  }
  obj.Prop("contestId", j.String(ExportId(team.Contest_id)))
  obj.Prop("isOpen", j.Boolean(team.Is_open))
  obj.Prop("isLocked", j.Boolean(team.Is_locked))
  obj.Prop("name", j.String(team.Name))
  obj.Prop("publicKey", j.String(team.Public_key))
  if v.teamId == team.Id {
    obj.Prop("accessCode", j.String(team.Access_code))
    v.teamMembers.Need(team.Id)
  }
  v.Add(fmt.Sprintf("teams %s", id), obj)
  return id
}

func (v *View) addTeamMember(member *model.TeamMember) string {
  id := fmt.Sprintf("%d_%d", member.Team_id, member.User_id)
  obj := j.Object()
  obj.Prop("teamId", j.String(ExportId(member.Team_id)))
  obj.Prop("userId", j.String(ExportId(member.User_id)))
  obj.Prop("joinedAt", j.String(member.Joined_at))
  obj.Prop("isCreator", j.Boolean(member.Is_creator))
  v.Add(fmt.Sprintf("teamMembers %s", id), obj)
  return id
}

func (v *View) addContest(contest *model.Contest) string {
  id := ExportId(contest.Id)
  obj := j.Object()
  obj.Prop("id", j.String(ExportId(contest.Id)))
  obj.Prop("createdAt", j.String(contest.Created_at))
  obj.Prop("updatedAt", j.String(contest.Updated_at))
  obj.Prop("title", j.String(contest.Title))
  obj.Prop("description", j.String(contest.Description))
  obj.Prop("logoUrl", j.String(contest.Logo_url))
  obj.Prop("taskId", j.String(ExportId(contest.Task_id)))
  obj.Prop("startsAt", j.String(contest.Starts_at))
  obj.Prop("endsAt", j.String(contest.Ends_at))
  v.Add(fmt.Sprintf("contests %s", id), obj)
  return id
}

func (v *View) addTask(task *model.Task) string {
  id := ExportId(task.Id)
  obj := j.Object()
  obj.Prop("id", j.String(ExportId(task.Id)))
  obj.Prop("title", j.String(task.Title))
  obj.Prop("createdAt", j.String(task.Created_at))
  obj.Prop("updatedAt", j.String(task.Updated_at))
  v.Add(fmt.Sprintf("tasks %s", id), obj)
  return id
}

func (v *View) addTaskResource(resource *model.TaskResource) string {
  id := ExportId(resource.Id)
  obj := j.Object()
  obj.Prop("id", j.String(ExportId(resource.Id)))
  obj.Prop("createdAt", j.String(resource.Created_at))
  obj.Prop("updatedAt", j.String(resource.Updated_at))
  obj.Prop("taskId", j.String(ExportId(resource.Task_id)))
  obj.Prop("rank", j.String(resource.Rank))
  obj.Prop("title", j.String(resource.Title))
  obj.Prop("description", j.String(resource.Description))
  obj.Prop("url", j.String(resource.Url))
  obj.Prop("html", j.String(resource.Html))
  v.Add(fmt.Sprintf("taskResources %s", id), obj)
  return id
}

func (v *View) addChain(chain *model.Chain) string {
  id := ExportId(chain.Id)
  obj := j.Object()
  obj.Prop("id", j.String(ExportId(chain.Id)))
  obj.Prop("createdAt", j.String(chain.Created_at))
  obj.Prop("updatedAt", j.String(chain.Updated_at))
  obj.Prop("contestId", j.String(ExportId(chain.Contest_id)))
  ownerId := j.Null
  if chain.Owner_id.Valid {
    ownerId = j.String(ExportId(chain.Owner_id.Int64))
  }
  obj.Prop("ownerId", ownerId)
  parentId := j.Null
  if chain.Parent_id.Valid {
    parentId = j.String(ExportId(chain.Parent_id.Int64))
  }
  obj.Prop("parentId", parentId)
  obj.Prop("statusId", j.String(ExportId(chain.Status_id))) // XXX
  obj.Prop("title", j.String(chain.Title))
  obj.Prop("description", j.String(chain.Description))
  obj.Prop("interfaceText", j.String(chain.Interface_text))
  obj.Prop("implementationText", j.String(chain.Implementation_text))
  obj.Prop("protocolHash", j.String(chain.Protocol_hash))
  obj.Prop("newProtocolHash", j.String(chain.New_protocol_hash))
  startedAt := j.Null
  if chain.Started_at.Valid {
    startedAt = j.String(chain.Started_at.String)
  }
  obj.Prop("startedAt", startedAt)
  obj.Prop("currentGameKey", j.String(chain.Game_key))
  obj.Prop("nbVotesApprove", j.Int(chain.Nb_votes_approve))
  obj.Prop("nbVotesReject", j.Int(chain.Nb_votes_reject))
  obj.Prop("nbVotesUnknown", j.Int(chain.Nb_votes_unknown))
  v.Add(fmt.Sprintf("chains %s", id), obj)
  return id
}
