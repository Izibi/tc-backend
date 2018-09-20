
package auth

import (
  ji "github.com/json-iterator/go"
)

/*
  type Profile = {
    "id": number,
    "login": string,
    "language": string,
    "first_name": string,
    "last_name": string,
    "timezone": string,
    "country_code": string,
    "picture": string,
    "has_picture": boolean,
    "last_login": string,
    "primary_email": string,
    "badges": Badge[],
  }
  type Badge = {
    "id": number,
    "url": string,
    "code": string | null,
    "do_not_possess": boolean,
    "data": string | null,
  }
*/

type userProfile struct {
  profile []byte
}

func LoadUserProfile(body []byte) *userProfile {
  return &userProfile{body}
}

func (p *userProfile) Id() string {
  return ji.Get(p.profile, "id").ToString()
}

func (p *userProfile) Username() string {
  return ji.Get(p.profile, "login").ToString()
}

func (p *userProfile) Firstname() string {
  return ji.Get(p.profile, "first_name").ToString()
}

func (p *userProfile) Lastname() string {
  return ji.Get(p.profile, "last_name").ToString()
}

func (p *userProfile) Badges() []string {
  iter := ji.ParseBytes(ji.ConfigDefault, p.profile)
  badges := []string{}
  var key1, key2 string
  for key1 = iter.ReadObject(); key1 != ""; key1 = iter.ReadObject() {
    if key1 == "badges" {
      for iter.ReadArray() {
        for key2 = iter.ReadObject(); key2 != ""; key2 = iter.ReadObject() {
          if key2 == "url" {
            badges = append(badges, iter.ReadString())
          } else {
            iter.Skip()
          }
        }
      }
    } else {
      iter.Skip()
    }
  }
  return badges
}
