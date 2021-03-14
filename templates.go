package main

import (
	"crypto/x509"
	"text/template"
	"time"
)

type Entry struct {
	ID        int
	Title     string
	URL       string
	Published time.Time
	Feed      string
}

type Feed struct {
	Title       string
	Description string
	URL         string
	Author      string
	Updated     time.Time
	VoteCnt     int
	VoteAmt     int
}

type Author struct {
	ID    int
	Name  string
	URL   string
	Email string
}

type DashboardPage struct {
	Feeds []*Feed
	Logo  string
}

var dashboardPage = template.Must(template.
	New("dashboard").
	Funcs(template.FuncMap{
		"date": func(date time.Time) string {
			return date.Format("Monday, January 2 2006")
		},
	}).
	Parse(`{{.Logo}}

=> /about About gemmit: the front page of gemini
=> /add Add a new feed
=> /earn How to earn

{{- if .Feeds }}
## Top 10 Feeds of all time
{{range .Feeds}}
=> {{.URL}} {{.Title}} - {{.Description}}
^ {{.VoteCnt}} ^ | ɱ {{.VoteAmt}} ɱ
Curated by {{.Author}} on {{.Updated | date}}
{{end}}
{{end}}


=> /top?t=all More
`))

type WelcomePage struct {
	Cert *x509.Certificate
	Hash string
	Logo string
}

var welcomePage = template.Must(template.
	New("welcome").
	Parse(`{{.Logo}}

Welcome to gemmit! We have registered your account automatically using your
client certificate.

Subject:
	{{.Cert.Subject.ToRDNSequence}}
SHA-256 Hash:
	{{.Hash}}

Keep this certificate safe! You'll need to back it up and bring it with you to
new devices in order to access your account. Your certificate hash is the only
information we store to identify you, and we cannot help you recover your
account if you lose your certificate.

Let's start by adding some feeds. You can add your own feed or any other hidden gems within Geminispace.

=> /add Add feed
`))

type AboutPage struct {
	Logo string
}

var aboutPage = template.Must(template.
	New("about").
	Parse(`{{.Logo}}

Gemmit is a social news aggregation and web content rating website for the gemini protocol. It enables discovery of highly rated Gemini feeds.

Gemmit seeks not to build a walled garden, but to leverage existing web protocols that respect security through decentralization and privacy.

Feeds are ranked via votes published on Monero.
Votes for the sake of gemmit are defined as: any incoming transaction to a blockchain address that has been associated with an atom feeds or entries respective author(s)

This approach has several benefits.
* immutability: votes can not be taken back
* censorship resistance: votes can not be prevented
* privacy: votes can not be tied back to an identity, unless voluntarily revealed
* monetary: votes act as micropayments to content producers
* composability: votes are not locked or owned by a central service
* auditability: votes can be publicly verified by all
* sovereignty: Content producers own both their content and the associated metadata for voting / payments

Gemmit is open source:

=> https://github.com/t-900-a/gemmit Gemmit on Github

=> / Back to the Feeds
`))

type EarnPage struct {
	Logo string
}

var earnPage = template.Must(template.
	New("Earn").
	Parse(`{{.Logo}}

Gemmit ranks feeds by their popularity. Popularity is measured by how much each feed author earns.
As a feed author your subscribers will reward you with micropayments that will increase your ranking on Gemmit.

## In order to start earning you will need to ...
* Create a wallet
=> https://mymonero.com/ Monero Wallet
* Add your payment address to your atom feed
* Add your secret view key to your atom feed
> <feed xmlns="http://www.w3.org/2005/Atom">
> 	<id>gemini://example.com/</id>
> 	<title>Example</title>
> 	<author>
> 		<name>anon</name>
> 		<content type="application/monero-viewkey">ba98e9501f1f5bc930e7c9fedb2424d17650da85936fb78b7d95ef1f3e306d02</content>
> 		<content type="application/monero-paymentrequest">monero:48uxJgXzhL78adi3i279zmQPcddMSam6mCX13JU2fLWcjeB2QhwPT9BMwNnrVcQGiXRDoHhtYz6Un8CpQ5LNHPTeJWJwsMT</content>
> 	</author>
> 	<entry>
> 		...
* Add your feed to Gemmit
=> /add Add feed

=> / Back to the Feeds
`))

var gemmitLogo = "\u0020.\u0020\u0020\u0020\u0020\u0020'\u0020\u0020\u0020\u0020\u0020,\n\u0020\u0020__G͟E͟M͟M͟I͟T͟__\n_\u0020/_|_____|_\\\u0020_\n\u0020\u0020'.\u0020\\\u0020\u0020\u0020/\u0020.'\n\u0020\u0020\u0020\u0020'.\\\u0020/.'\n\u0020\u0020\u0020\u0020\u0020\u0020'.'\n"
