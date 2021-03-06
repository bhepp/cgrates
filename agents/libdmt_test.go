/*
Real-time Charging System for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

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

package agents

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
	"time"

	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/utils"
	"github.com/fiorix/go-diameter/diam"
	"github.com/fiorix/go-diameter/diam/avp"
	"github.com/fiorix/go-diameter/diam/datatype"
	"github.com/fiorix/go-diameter/diam/dict"
)

func TestDisectUsageForCCR(t *testing.T) {
	if reqType, reqNr, reqCCTime, usedCCTime := disectUsageForCCR(time.Duration(0)*time.Second, time.Duration(300)*time.Second, false); reqType != 1 || reqNr != 0 || reqCCTime != 300 || usedCCTime != 0 {
		t.Error(reqType, reqNr, reqCCTime, usedCCTime)
	}
	if reqType, reqNr, reqCCTime, usedCCTime := disectUsageForCCR(time.Duration(35)*time.Second, time.Duration(300)*time.Second, false); reqType != 2 || reqNr != 0 || reqCCTime != 300 || usedCCTime != 35 {
		t.Error(reqType, reqNr, reqCCTime, usedCCTime)
	}
	if reqType, reqNr, reqCCTime, usedCCTime := disectUsageForCCR(time.Duration(935)*time.Second, time.Duration(300)*time.Second, false); reqType != 2 || reqNr != 3 || reqCCTime != 300 || usedCCTime != 35 {
		t.Error(reqType, reqNr, reqCCTime, usedCCTime)
	}
	if reqType, reqNr, reqCCTime, usedCCTime := disectUsageForCCR(time.Duration(35)*time.Second, time.Duration(300)*time.Second, true); reqType != 3 || reqNr != 1 || reqCCTime != 0 || usedCCTime != 35 {
		t.Error(reqType, reqNr, reqCCTime, usedCCTime)
	}
	if reqType, reqNr, reqCCTime, usedCCTime := disectUsageForCCR(time.Duration(610)*time.Second, time.Duration(300)*time.Second, true); reqType != 3 || reqNr != 3 || reqCCTime != 0 || usedCCTime != 10 {
		t.Error(reqType, reqNr, reqCCTime, usedCCTime)
	}
	if reqType, reqNr, reqCCTime, usedCCTime := disectUsageForCCR(time.Duration(935)*time.Second, time.Duration(300)*time.Second, true); reqType != 3 || reqNr != 4 || reqCCTime != 0 || usedCCTime != 35 {
		t.Error(reqType, reqNr, reqCCTime, usedCCTime)
	}
}

func TestUsageFromCCR(t *testing.T) {
	if usage := usageFromCCR(1, 0, 300, 0, time.Duration(300)*time.Second); usage != time.Duration(300)*time.Second {
		t.Error(usage)
	}
	if usage := usageFromCCR(2, 0, 300, 300, time.Duration(300)*time.Second); usage != time.Duration(300)*time.Second {
		t.Error(usage)
	}
	if usage := usageFromCCR(2, 3, 300, 300, time.Duration(300)*time.Second); usage != time.Duration(300)*time.Second {
		t.Error(usage.Seconds())
	}
	if usage := usageFromCCR(3, 3, 0, 10, time.Duration(300)*time.Second); usage != time.Duration(610)*time.Second {
		t.Error(usage)
	}
	if usage := usageFromCCR(3, 4, 0, 35, time.Duration(300)*time.Second); usage != time.Duration(935)*time.Second {
		t.Error(usage)
	}
	if usage := usageFromCCR(3, 1, 0, 35, time.Duration(300)*time.Second); usage != time.Duration(35)*time.Second {
		t.Error(usage)
	}
	if usage := usageFromCCR(1, 0, 360, 0, time.Duration(360)*time.Second); usage != time.Duration(360)*time.Second {
		t.Error(usage)
	}
}

func TestAvpValAsString(t *testing.T) {
	originHostStr := "unit_test"
	a := diam.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity(originHostStr))
	if avpValStr := avpValAsString(a); avpValStr != originHostStr {
		t.Errorf("Expected: %s, received: %s", originHostStr, avpValStr)
	}
}

func TestFieldOutVal(t *testing.T) {
	m := diam.NewRequest(diam.CreditControl, 4, nil)
	m.NewAVP("Session-Id", avp.Mbit, 0, datatype.UTF8String("simuhuawei;1449573472;00002"))
	m.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(0)),             // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("33708000003")), // Subscription-Id-Data
		}})
	m.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(1)),              // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("208708000003")), // Subscription-Id-Data
		}})
	m.NewAVP("Service-Identifier", avp.Mbit, 0, datatype.Unsigned32(0))
	m.NewAVP("Requested-Service-Unit", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(420, avp.Mbit, 0, datatype.Unsigned32(360))}}) // CC-Time
	cfgFld := &config.CfgCdrField{Tag: "StaticTest", Type: utils.META_COMPOSED, FieldId: utils.TOR,
		Value: utils.ParseRSRFieldsMustCompile("^*voice", utils.INFIELD_SEP), Mandatory: true}
	eOut := "*voice"
	if fldOut, err := fieldOutVal(m, cfgFld, time.Duration(0)); err != nil {
		t.Error(err)
	} else if fldOut != eOut {
		t.Errorf("Expecting: %s, received: %s", eOut, fldOut)
	}
	cfgFld = &config.CfgCdrField{Tag: "ComposedTest", Type: utils.META_COMPOSED, FieldId: utils.DESTINATION,
		Value: utils.ParseRSRFieldsMustCompile("Requested-Service-Unit>CC-Time", utils.INFIELD_SEP), Mandatory: true}
	eOut = "360"
	if fldOut, err := fieldOutVal(m, cfgFld, time.Duration(0)); err != nil {
		t.Error(err)
	} else if fldOut != eOut {
		t.Errorf("Expecting: %s, received: %s", eOut, fldOut)
	}
	// Without filter, we shoud get always the first subscriptionId
	cfgFld = &config.CfgCdrField{Tag: "Grouped1", Type: utils.MetaGrouped, FieldId: "Account",
		Value: utils.ParseRSRFieldsMustCompile("Subscription-Id>Subscription-Id-Data", utils.INFIELD_SEP), Mandatory: true}
	eOut = "33708000003"
	if fldOut, err := fieldOutVal(m, cfgFld, time.Duration(0)); err != nil {
		t.Error(err)
	} else if fldOut != eOut {
		t.Errorf("Expecting: %s, received: %s", eOut, fldOut)
	}
	// Without groupedAVP, we shoud get the first subscriptionId
	cfgFld = &config.CfgCdrField{Tag: "Grouped2", Type: utils.MetaGrouped, FieldId: "Account",
		FieldFilter: utils.ParseRSRFieldsMustCompile("Subscription-Id>Subscription-Id-Type(1)", utils.INFIELD_SEP),
		Value:       utils.ParseRSRFieldsMustCompile("Subscription-Id>Subscription-Id-Data", utils.INFIELD_SEP), Mandatory: true}
	eOut = "208708000003"
	if fldOut, err := fieldOutVal(m, cfgFld, time.Duration(0)); err != nil {
		t.Error(err)
	} else if fldOut != eOut {
		t.Errorf("Expecting: %s, received: %s", eOut, fldOut)
	}
}

func TestSerializeAVPValueFromString(t *testing.T) {
	dictAVP, _ := dict.Default.FindAVP(4, "Session-Id")
	eValByte := []byte("simuhuawei;1449573472;00002")
	if valByte, err := serializeAVPValueFromString(dictAVP, "simuhuawei;1449573472;00002", "UTC"); err != nil {
		t.Error(err)
	} else if !bytes.Equal(eValByte, valByte) {
		t.Errorf("Expecting: %+v, received: %+v", eValByte, valByte)
	}
	dictAVP, _ = dict.Default.FindAVP(4, "Result-Code")
	eValByte = make([]byte, 4)
	binary.BigEndian.PutUint32(eValByte, uint32(5031))
	if valByte, err := serializeAVPValueFromString(dictAVP, "5031", "UTC"); err != nil {
		t.Error(err)
	} else if !bytes.Equal(eValByte, valByte) {
		t.Errorf("Expecting: %+v, received: %+v", eValByte, valByte)
	}
}

func TestMessageSetAVPsWithPath(t *testing.T) {
	eMessage := diam.NewRequest(diam.CreditControl, 4, nil)
	eMessage.NewAVP("Session-Id", avp.Mbit, 0, datatype.UTF8String("simuhuawei;1449573472;00002"))
	m := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 4, eMessage.Header.HopByHopID, eMessage.Header.EndToEndID, nil)
	if err := messageSetAVPsWithPath(m, []interface{}{"Session-Id", "Unknown"}, "simuhuawei;1449573472;00002", false, "UTC"); err == nil || err.Error() != "Could not find AVP Unknown" {
		t.Error(err)
	}
	if err := messageSetAVPsWithPath(m, []interface{}{"Session-Id"}, "simuhuawei;1449573472;00002", false, "UTC"); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(eMessage, m) {
		t.Errorf("Expecting: %+v, received: %+v", eMessage, m)
	}
	// test append
	eMessage.NewAVP("Session-Id", avp.Mbit, 0, datatype.UTF8String("simuhuawei;1449573472;00003"))
	if err := messageSetAVPsWithPath(m, []interface{}{"Session-Id"}, "simuhuawei;1449573472;00003", true, "UTC"); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(eMessage, m) {
		t.Errorf("Expecting: %+v, received: %+v", eMessage, m)
	}
	// test overwrite
	eMessage = diam.NewRequest(diam.CreditControl, 4, nil)
	eMessage.NewAVP("Session-Id", avp.Mbit, 0, datatype.UTF8String("simuhuawei;1449573472;00002"))
	m = diam.NewMessage(diam.CreditControl, diam.RequestFlag, 4, eMessage.Header.HopByHopID, eMessage.Header.EndToEndID, nil)
	if err := messageSetAVPsWithPath(m, []interface{}{"Session-Id"}, "simuhuawei;1449573472;00001", false, "UTC"); err != nil {
		t.Error(err)
	}
	if err := messageSetAVPsWithPath(m, []interface{}{"Session-Id"}, "simuhuawei;1449573472;00002", false, "UTC"); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(eMessage, m) {
		t.Errorf("Expecting: %+v, received: %+v", eMessage, m)
	}
	eMessage = diam.NewRequest(diam.CreditControl, 4, nil)
	eMessage.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("33708000003")), // Subscription-Id-Data
		}})
	m = diam.NewMessage(diam.CreditControl, diam.RequestFlag, 4, eMessage.Header.HopByHopID, eMessage.Header.EndToEndID, nil)
	if err := messageSetAVPsWithPath(m, []interface{}{"Subscription-Id", "Subscription-Id-Data"}, "33708000003", false, "UTC"); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(eMessage, m) {
		t.Errorf("Expecting: %+v, received: %+v", eMessage, m)
	}
	// test append
	eMessage.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(0)), // Subscription-Id-Data
		}})
	if err := messageSetAVPsWithPath(m, []interface{}{"Subscription-Id", "Subscription-Id-Type"}, "0", true, "UTC"); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(eMessage, m) {
		t.Errorf("Expecting: %+v, received: %+v", eMessage, m)
	}
	// test group append
	eMessage = diam.NewRequest(diam.CreditControl, 4, nil)
	eMessage.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(0)),             // Subscription-Id-Data
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("33708000003")), // Subscription-Id-Data
		}})
	eMsgSrl, _ := eMessage.Serialize()
	m = diam.NewMessage(diam.CreditControl, diam.RequestFlag, 4, eMessage.Header.HopByHopID, eMessage.Header.EndToEndID, nil)
	if err := messageSetAVPsWithPath(m, []interface{}{"Subscription-Id", "Subscription-Id-Type"}, "0", false, "UTC"); err != nil {
		t.Error(err)
	}
	if err := messageSetAVPsWithPath(m, []interface{}{"Subscription-Id", "Subscription-Id-Data"}, "33708000003", false, "UTC"); err != nil {
		t.Error(err)
	} else {
		mSrl, _ := m.Serialize()
		if !bytes.Equal(eMsgSrl, mSrl) {
			t.Errorf("Expecting: %+v, received: %+v", eMessage, m)
		}
	}
	eMessage = diam.NewRequest(diam.CreditControl, 4, nil)
	eMessage.NewAVP("Granted-Service-Unit", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(420, avp.Mbit, 0, datatype.Unsigned32(300)), // Subscription-Id-Data
		}})
	m = diam.NewMessage(diam.CreditControl, diam.RequestFlag, 4, eMessage.Header.HopByHopID, eMessage.Header.EndToEndID, nil)
	if err := messageSetAVPsWithPath(m, []interface{}{"Granted-Service-Unit", "CC-Time"}, "300", false, "UTC"); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(eMessage, m) {
		t.Errorf("Expecting: %+v, received: %+v", eMessage, m)
	}
}

func TestCCASetProcessorAVPs(t *testing.T) {
	ccr := &CCR{ // Bare information, just the one needed for answer
		SessionId:         "routinga;1442095190;1476802709",
		AuthApplicationId: 4,
		CCRequestType:     1,
		CCRequestNumber:   0,
	}
	ccr.diamMessage = ccr.AsBareDiameterMessage()
	ccr.diamMessage.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(0)),             // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("33708000003")), // Subscription-Id-Data
		}})
	ccr.debitInterval = time.Duration(300) * time.Second
	cca := NewBareCCAFromCCR(ccr, "CGR-DA", "cgrates.org")
	reqProcessor := &config.DARequestProcessor{Id: "UNIT_TEST", // Set template for tests
		CCAFields: []*config.CfgCdrField{
			&config.CfgCdrField{Tag: "Subscription-Id/Subscription-Id-Type", Type: utils.META_COMPOSED,
				FieldId: "Subscription-Id>Subscription-Id-Type",
				Value:   utils.ParseRSRFieldsMustCompile("Subscription-Id>Subscription-Id-Type", utils.INFIELD_SEP), Mandatory: true},
			&config.CfgCdrField{Tag: "Subscription-Id/Subscription-Id-Data", Type: utils.META_COMPOSED,
				FieldId: "Subscription-Id>Subscription-Id-Data",
				Value:   utils.ParseRSRFieldsMustCompile("Subscription-Id>Subscription-Id-Data", utils.INFIELD_SEP), Mandatory: true},
		},
	}
	eMessage := cca.AsDiameterMessage()
	eMessage.NewAVP("Subscription-Id", avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(0)),             // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String("33708000003")), // Subscription-Id-Data
		}})
	if err := cca.SetProcessorAVPs(reqProcessor, 0); err != nil {
		t.Error(err)
	} else if ccaMsg := cca.AsDiameterMessage(); !reflect.DeepEqual(eMessage, ccaMsg) {
		t.Errorf("Expecting: %+v, received: %+v", eMessage, ccaMsg)
	}
}
