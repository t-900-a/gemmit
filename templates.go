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
	VoteAmt     string
}

type Author struct {
	ID    int
	Name  string
	URL   string
	Email string
}

type DashboardPage struct {
	Feeds   []*Feed
	Logo    string
	Newline string
}

var dashboardPage = template.Must(template.
	New("dashboard").
	Funcs(template.FuncMap{
		"date": func(date time.Time) string {
			return date.Format("Monday, January 2 2006")
		},
	}).
	Parse(`{{.Logo}}

=> /about About Gemmit: the front page of gemini
=> /add Add a new feed
=> /earn How to earn
=> /vote How to vote
{{.Newline}}
{{- if .Feeds }}
## Top 10 Feeds of all time
{{range .Feeds}}
=> {{.URL}} {{.Title}} - {{.Description}}
Votes: {{.VoteCnt}} | ɱ  {{.VoteAmt}}
Last updated by {{.Author}} on {{.Updated | date}}
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

=> https://github.com/t-900-a/gemmit Gemmit is free & open source
=> https://github.com/t-900-a/awesome-gemmit/ Gemmit is an ecosystem

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
# Create a wallet
=> https://mymonero.com/ Monero Wallet
# Add your payment address to your atom feed (as a subelement of author)
> <atom:link rel="payment" type="application/monero-paymentrequest" href="monero:donate.getmonero.org"/>
# Add your secret view key to your atom feed (as a subelement of author)
>   ...
> 	<author>
> 		<name>anon</name>
> 		<content type="application/monero-viewkey">ba98e9501f1f5bc930e7c9fedb2424d17650da85936fb78b7d95ef1f3e306d02</content>
> 	</author>
> 	...
# Request donations within the content that you produce
=> http://asciiqr.com/ Generate ASCII QR Code Online
=> https://github.com/fumiyas/qrc Or locally
█▀▀▀▀▀█ ▄ █ ▀▀▄▄▀ █▀▀▀▀▀█
█ ███ █ ▄▄▄ █▄███ █ ███ █
█ ▀▀▀ █ ▄█ ▄▄█▄▄█ █ ▀▀▀ █
▀▀▀▀▀▀▀ ▀▄█▄▀ █▄█ ▀▀▀▀▀▀▀
█████▄▀▀█▀▄▀██▀▀▄▀ █ ▀▄▀ 
▀█ ▀▀▄▀ ▄ ▀▄ ▀▀▀█▀ ██▀▀█▀
█▀ █ █▀██▀▀▄▀▀▀▀▄▀▀▄▀▀▀▀ 
█  ▀ ▀▀ ▄▀▄▄█ █▀▄▄█▄   ██
▀ ▀ ▀▀▀ ▄█▄▄ ▀█▄█▀▀▀█▄▀▀ 
█▀▀▀▀▀█ ▀▀█▄ ▀ ▄█ ▀ █▀▀█▀
█ ███ █ ██▄█▀█▀▀▀██▀█▀█▄▄
█ ▀▀▀ █ ██▀▄▀▀█▄▄█▄▀▄▄▀ █
▀▀▀▀▀▀▀ ▀▀ ▀▀▀    ▀ ▀ ▀▀▀
=> monero:donate.getmonero.org Donate for more great content
# Lastly, add your feed to Gemmit
=> /add Add feed

=> https://github.com/t-900-a/awesome-gemmit/ As the ecosystem grows scripts to automate feed generation will become available

=> / Back to the Feeds
`))

type VotePage struct {
	Logo string
}

var votePage = template.Must(template.
	New("Vote").
	Parse(`{{.Logo}}

Gemmit ranks feeds by their popularity.

Subscribe to any feed on the site, if you enjoy the feeds content please vote/tip using Monero.

Their address should be displayed within in their content as a QR Code or a link. After tipping your vote will be reflected on this site.

Feeds can be subscribed to via a dedicated feed reader, but most Gemini Browsers include functionality to subscribe to feeds.

=> https://github.com/t-900-a/awesome-gemmit/#readers More information on readers


=> / Back to the Feeds
`))

var gemmitLogo = "```\u0020.\u0020\u0020\u0020\u0020\u0020'\u0020\u0020\u0020\u0020,\n\u0020\u0020__G͟E͟M͟M͟I͟T͟__\n_\u0020/_|_____|_\\\u0020_\n\u0020\u0020'.\u0020\\\u0020\u0020\u0020/\u0020.'\n\u0020\u0020\u0020\u0020'.\\\u0020/.'\n\u0020\u0020\u0020\u0020\u0020\u0020'.'\n```"

type AcceptedPayment struct {
	PayType    string
	ViewKey    string
	Address    string
	Registered bool
}
