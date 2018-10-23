
package view

import (
  "fmt"
  "sort"
  "strconv"
  "time"
  "github.com/go-sql-driver/mysql"
  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/model"
)

type View struct {
  model *model.Model

  userId int64
  teamId int64
  contestId int64
  isAdmin bool

  tasks LoadSet
  users LoadSet
  teams LoadSet
  teamMembers LoadSet

  result j.IObject
  entities map[string]j.Value
}

func New(model *model.Model) *View {
  return &View{
    model: model,
    result: j.Object(),
    entities: make(map[string]j.Value),
  }
}

func (v *View) Set(key string, value j.Value) {
  v.result.Prop(key, value)
}

func (v *View) Add(key string, view j.Value) {
  v.entities[key] = view
}

func (v *View) Has(key string) bool {
  _, ok := v.entities[key]
  return ok
}

func (v *View) Flat() j.Value {
  res := j.Object()
  res.Prop("result", v.result)
  entities := j.Object()
  for _, key := range orderedMapKeys(v.entities) {
    entities.Prop(key, v.entities[key])
  }
  res.Prop("entities", entities)
  return res
}

func orderedMapKeys(m map[string]j.Value) []string {
  keys := make([]string, len(m))
  i := 0
  for key := range m {
    keys[i] = key
    i++
  }
  sort.Strings(keys)
  return keys
}


func ImportId(id string) int64 {
  n, err := strconv.ParseInt(id, 10, 64)
  if err != nil { return 0 }
  return n
}

func ExportId(id int64) string {
  return strconv.FormatInt(id, 10)
}

func timeProp(obj j.IObject, key string, val time.Time) {
  obj.Prop(key, j.String(val.Format(time.RFC3339)))
}

func nullTimeProp(obj j.IObject, key string, val mysql.NullTime) {
  if val.Valid {
    obj.Prop(key, j.String(val.Time.Format(time.RFC3339)))
  } else {
    obj.Prop(key, j.Null)
  }
}

func (v *View) loadTeams(ids []int64) error {
  teams, err := v.model.LoadTeamsById(ids)
  if err != nil { return err }
  for i := range teams {
    v.addTeam(&teams[i])
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
      ids.Item(j.String(fmt.Sprintf("%d_%d", teamId, member.User_id)))
      v.addTeamMember(member)
    }
    obj := j.Object()
    obj.Prop("memberIds", ids)
    v.Add(fmt.Sprintf("teams#members %s", ExportId(teamId)), obj)
  }
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

/*** Add functions ***/

func (v *View) addUser(user *model.User) {
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
  v.Add(fmt.Sprintf("users %s", ExportId(user.Id)), obj)
}

func (v *View) addTeam(team *model.Team) {
  id := ExportId(team.Id)
  obj := j.Object()
  obj.Prop("id", j.String(id))
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
}

func (v *View) addTeamMember(member *model.TeamMember) {
  obj := j.Object()
  obj.Prop("teamId", j.String(ExportId(member.Team_id)))
  obj.Prop("userId", j.String(ExportId(member.User_id)))
  obj.Prop("joinedAt", j.String(member.Joined_at))
  obj.Prop("isCreator", j.Boolean(member.Is_creator))
  v.Add(fmt.Sprintf("teamMembers %d_%d", member.Team_id, member.User_id), obj)
}

func (v *View) addContest(contest *model.Contest) {
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
  v.Add(fmt.Sprintf("contests %s", ExportId(contest.Id)), obj)
}

func (v *View) addTask(task *model.Task) {
  obj := j.Object()
  obj.Prop("id", j.String(ExportId(task.Id)))
  obj.Prop("title", j.String(task.Title))
  obj.Prop("createdAt", j.String(task.Created_at))
  obj.Prop("updatedAt", j.String(task.Updated_at))
  v.Add(fmt.Sprintf("tasks %s", ExportId(task.Id)), obj)
}

func (v *View) addTaskResource(resource *model.TaskResource) {
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
  v.Add(fmt.Sprintf("taskResources %s", ExportId(resource.Id)), obj)
}

func (v *View) addChain(chain *model.Chain) {
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
  v.Add(fmt.Sprintf("chains %s", ExportId(chain.Id)), obj)
}

func (v *View) addGamePlayer(player *model.GamePlayer) {
  obj := j.Object()
  obj.Prop("gameId", j.Int64(player.Game_id))
  obj.Prop("rank", j.Uint32(player.Rank))
  obj.Prop("teamId", j.Int64(player.Team_id))
  obj.Prop("botId", j.Uint32(player.Team_player /* TODO res.Bot_id */))
  obj.Prop("createdAt", j.Time(player.Created_at))
  obj.Prop("updatedAt", j.Time(player.Updated_at))
  obj.Prop("commands", j.Raw(player.Commands))
  v.Add(fmt.Sprintf("game_players %s.%d",
    ExportId(player.Game_id), player.Rank), obj)
}

func ViewGame(game *model.Game) j.Value {
  if game == nil {
    return j.Null
  }
  view := j.Object()
  view.Prop("key", j.String(game.Game_key))
  view.Prop("createdAt", j.Time(game.Created_at))
  view.Prop("updatedAt", j.Time(game.Updated_at))
  view.Prop("ownerId", j.String(ExportId(game.Owner_id)))
  view.Prop("firstBlock", j.String(game.First_block))
  view.Prop("lastBlock", j.String(game.Last_block))
  nullTimeProp(view, "startedAt", game.Started_at)
  nullTimeProp(view, "roundEndsAt", game.Round_ends_at)
  view.Prop("isLocked", j.Boolean(game.Locked))
  view.Prop("currentRound", j.Uint64(game.Current_round))
  view.Prop("nbCyclesPerRound", j.Uint(game.Nb_cycles_per_round))
  return view
}
