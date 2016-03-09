package pagerduty

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"time"
)

// API docs: https://developer.pagerduty.com/documentation/rest/escalation_policies/on_call

// keep an in-memory cache that expires after an hour
var policyCache struct {
	policies  []EscalationPolicy
	timestamp time.Time
}

type EscalationPolicyResponse struct {
	EscalationPolicies []EscalationPolicy `json:"escalation_policies"`
	Limit              int                `json:"limit"`
	Offset             int                `json:"offset"`
	Total              int                `json:"total"`
}

type EscalationPolicy struct {
	Id              string           `json:"id"`
	Name            string           `json:"name"`
	EscalationRules []EscalationRule `json:"escalation_rules"`
	Services        []Service        `json:"services"`
	OnCall          []OnCall         `json:"on_call"`
	NumLoops        int              `json:"num_loops"`
}

type EscalationRule struct {
	Id                       string     `json:"id"`
	EscalationDelayInMinutes int        `json:"escalation_delay_in_minutes"`
	RuleObject               RuleObject `json:"rule_object"`
}

type RuleObject struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Email    string `json:"email"`
	Timezone string `json:"time_zone"`
	Color    string `json:"color"`
}

type Service struct {
	Id                 string `json:"id"`
	Name               string `json:"name"`
	IntegrationEmail   string `json:"integration_email"`
	HtmlUrl            string `json:"html_url"`
	EscalationPolicyId string `json:"escalation_policy_id"`
}

func IsPolicyCached() bool {
	if policyCache.policies == nil {
		return false
	}
	return true
}

func GetEscalationPolicies(token, domain string) ([]EscalationPolicy, error) {
	if policyCache.policies != nil {
		now := time.Now()
		// expires every hour
		// TODO: make this configurable
		if now.Before(policyCache.timestamp.Add(time.Hour)) {
			log.Println("returning cached policies")
			return policyCache.policies, nil
		}
	}

	policies := make([]EscalationPolicy, 0)
	epresp := EscalationPolicyResponse{}
	offset := 0
	limit := 100

	for {
		url := pagedUrl("/api/v1/escalation_policies/on_call", domain, offset, limit)

		resp, err := authenticatedGet(url, token, "")
		if err != nil {
			log.Printf("GET %s failed: %s", url, err)
			return policies, err
		}

		data, err := ioutil.ReadAll(resp.Body)
		log.Printf("Got %d bytes from URL %q", len(data), url)

		err = json.Unmarshal(data, &epresp)
		if err != nil {
			log.Printf("json.Unmarshal failed: %s", err)
			return policies, err
		}

		policies = append(policies, epresp.EscalationPolicies...)

		// check for paging and bump the offset if needed
		if epresp.Offset < epresp.Total {
			offset = offset + limit
		} else {
			break
		}
	}

	// cache the data
	policyCache.policies = policies
	policyCache.timestamp = time.Now()

	return policies, nil
}