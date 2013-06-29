/*
Rating system designed to be used in VoIP Carriers World
Copyright (C) 2013 ITsysCOM

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package apier

import (
	"errors"
	"fmt"
	"github.com/cgrates/cgrates/rater"
	"github.com/cgrates/cgrates/scheduler"
	"github.com/cgrates/cgrates/utils"
	"log"
)

type Apier struct {
	StorDb rater.DataStorage
	Getter rater.DataStorage
	Sched  *scheduler.Scheduler
}

type AttrDestination struct {
	Id       string
	Prefixes []string
}

func (self *Apier) GetDestination(attr *AttrDestination, reply *AttrDestination) error {
	if dst, err := self.Getter.GetDestination(attr.Id); err != nil {
		return errors.New(utils.ERR_NOT_FOUND)
	} else {
		reply.Id = dst.Id
		reply.Prefixes = dst.Prefixes
	}
	return nil
}

func (self *Apier) SetDestination(attr *AttrDestination, reply *float64) error {
	d := &rater.Destination{
		Id:       attr.Id,
		Prefixes: attr.Prefixes,
	}
	if err := self.Getter.SetDestination(d); err != nil {
		return err
	}
	return nil
}

type AttrGetBalance struct {
	Tenant    string
	Account   string
	BalanceId string
	Direction string
}

// Get balance
func (self *Apier) GetBalance(attr *AttrGetBalance, reply *float64) error {
	tag := fmt.Sprintf("%s:%s:%s", attr.Direction, attr.Tenant, attr.Account)
	userBalance, err := self.Getter.GetUserBalance(tag)
	if err != nil {
		return err
	}

	if attr.Direction == "" {
		attr.Direction = rater.OUTBOUND
	}

	if balance, balExists := userBalance.BalanceMap[attr.BalanceId+attr.Direction]; !balExists {
		*reply = 0.0
	} else {
		*reply = balance
	}
	return nil
}

type AttrAddBalance struct {
	Tenant    string
	Account   string
	BalanceId string
	Direction string
	Value     float64
}

func (self *Apier) AddBalance(attr *AttrAddBalance, reply *float64) error {
	// what storage instance do we use?
	tag := fmt.Sprintf("%s:%s:%s", attr.Direction, attr.Tenant, attr.Account)

	if _, err := self.Getter.GetUserBalance(tag); err != nil {
		// create user balance if not exists
		ub := &rater.UserBalance{
			Id: tag,
		}
		if err := self.Getter.SetUserBalance(ub); err != nil {
			return err
		}
	}

	at := &rater.ActionTiming{
		UserBalanceIds: []string{tag},
	}

	if attr.Direction == "" {
		attr.Direction = rater.OUTBOUND
	}

	at.SetActions(rater.Actions{&rater.Action{ActionType: rater.TOPUP, BalanceId: attr.BalanceId, Direction: attr.Direction, Units: attr.Value}})

	if err := at.Execute(); err != nil {
		return err
	}
	// what to put in replay?
	return nil
}

type AttrExecuteAction struct {
	Direction string
	Tenant    string
	Account   string
	BalanceId string
	ActionsId string
}

func (self *Apier) ExecuteAction(attr *AttrExecuteAction, reply *float64) error {
	tag := fmt.Sprintf("%s:%s:%s", attr.Direction, attr.Tenant, attr.Account)
	at := &rater.ActionTiming{
		UserBalanceIds: []string{tag},
		ActionsId:      attr.ActionsId,
	}

	if err := at.Execute(); err != nil {
		return err
	}
	// what to put in replay
	return nil
}

type AttrSetRatingProfile struct {
	Direction     string
	Tenant        string
	TOR           string
	Subject       string
	SourceSubject string
}

func (self *Apier) SetRatingProfile(attr *AttrSetRatingProfile, reply *float64) error {
	cd := &rater.CallDescriptor{
		Direction: attr.Direction,
		Tenant:    attr.Tenant,
		TOR:       attr.TOR,
		Subject:   attr.Subject,
	}
	destination, err := self.Getter.GetRatingProfile(cd.GetKey())
	log.Print("1: " + cd.GetKey())
	if err != nil {
		destination = &rater.RatingProfile{Id: cd.GetKey()}
	}
	cd.Subject = attr.SourceSubject
	source, err := self.Getter.GetRatingProfile(cd.GetKey())
	log.Print("2: " + cd.GetKey())
	if err != nil {
		log.Print("here: " + err.Error())
		return err
	}
	destination.DestinationMap = source.DestinationMap
	destination.FallbackKey = source.FallbackKey
	err = self.Getter.SetRatingProfile(destination)
	return err
}

type AttrActionTrigger struct {
	Tenant         string
	Account        string
	Direction      string
	BalanceId      string
	ThresholdValue float64
	DestinationId  string
	Weight         float64
	ActionsId      string
}

func (self *Apier) AddTriggeredAction(attr *AttrActionTrigger, reply *float64) error {
	if attr.Direction == "" {
		attr.Direction = rater.OUTBOUND
	}

	at := &rater.ActionTrigger{
		Id:             utils.GenUUID(),
		BalanceId:      attr.BalanceId,
		Direction:      attr.Direction,
		ThresholdValue: attr.ThresholdValue,
		DestinationId:  attr.DestinationId,
		Weight:         attr.Weight,
		ActionsId:      attr.ActionsId,
		Executed:       false,
	}

	tag := fmt.Sprintf("%s:%s:%s", attr.Direction, attr.Tenant, attr.Account)
	var dbErr error
	rater.AccLock.Guard(tag, func() (float64, error) {
		userBalance, err := self.Getter.GetUserBalance(tag)
		if err != nil {
			dbErr = err
			return 0, err
		}

		userBalance.ActionTriggers = append(userBalance.ActionTriggers, at)

		if err = self.Getter.SetUserBalance(userBalance); err != nil {
			dbErr = err
			return 0, err
		}
		return 0, nil
	})

	return dbErr
}

type AttrAccount struct {
	Tenant          string
	Direction       string
	Account         string
	Type            string // prepaid-postpaid
	ActionTimingsId string
}

func (self *Apier) AddAccount(attr *AttrAccount, reply *float64) error {
	tag := fmt.Sprintf("%s:%s:%s", attr.Direction, attr.Tenant, attr.Account)
	ub := &rater.UserBalance{
		Id:   tag,
		Type: attr.Type,
	}
	if err := self.Getter.SetUserBalance(ub); err != nil {
		return err
	}
	if attr.ActionTimingsId != "" {
		if ats, err := self.Getter.GetActionTimings(attr.ActionTimingsId); err == nil {
			for _, at := range ats {
				at.UserBalanceIds = append(at.UserBalanceIds, tag)
			}
			self.Getter.SetActionTimings(attr.ActionTimingsId, ats)
			if self.Sched != nil {
				self.Sched.LoadActionTimings(self.Getter)
				self.Sched.Restart()
			}
		} else {
			return err
		}
	}
	return nil
}