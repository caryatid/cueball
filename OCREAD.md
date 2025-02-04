
# Learning is FUNdamental
Nearly every fundamental process here is different than I've seen elsewhere. Many 
are directly at odds with what I've seen. Despite the fact other orgs have had
staggering amounts of waste and foolishness; the worst (one exception) of them is like a wee little
candle to the sun of absurdity shown brightly by the methodologies in place at OC.

I want to underscore as boldly as possible that the circumstances below have 
been effectively ubiquitous; OC is a complete outlier and utterly unique in my 20 years.
I'm ok with marching to the beat of your own drum provided the results
are at or above other known methods. That is not the case here. Other than one
position I had with a federal agency I've never seen such waste and inefficency. 

## deployment environments
### basic definitions
#### Environments

    - Production: Artifact only deployments. Isloated: any manual interaction *at all* is an "incident" or is becuase of an "incident". No exceptions at all. If you need manual interaction to deploy software than you, definitionally, do not have a production system and all deployments ought be hard halted until you do. Can be ephemerally deployed using disaster recovery data for population. Previous releases (rollbacks) are completely reliable and evidenced - but not necessarily easy b/c of the data.
    - UAT: Artifact only deployments. Isoloated: no manual interaction but no incident if there is provided UAT is ultimatlely still entirely created by automation. UAT is identical to production on moment of prod deploy and may be one version ahead for any deployable at other times. Can be deployed ephemerally (automatically: push button) other than the actual data itself. Any new UAT deployment is, definitionally, a staging environment. UAT has "real" data although it may be obfuscated for PII reasons. That data difference and the size of the environment (number of machines) are the only allowed constant deviations from PROD. rollbacks push button and reliable -- should be easy.
    - Staging: Artifact only deployments. Manual interaction allowed, but when manual interaction occurs it ought trigger a full redploy of staging before next prod release. Can have all kinds of mixmatched versions but generally is like a UAT that is smaller and devs/QA/ops will broadly have access to the data and backing services.  Putting it in a broken state is allowed if everyone involved is aware. Generally, however, if an action risks breakage then you ought simply deploy a new staging and work there. Can deploy an previous release, but not done as rollback -- just a new deploy with older software versions. -- easy because no data reqs other than schema and all artifact deploy.
    - Development: Can be deployed to directly by engineering staff. Is easily and individually ephemeral. Meaning *any* dev can say (ideally to an RPC or REST endpoint) "give me a new dev env" and get one quickly by an automated process. Additionally such environments can have any versions for any deployed components.  The initial deployment of a dev environment will be pushbutton and artifact only; but further updates can be willy nilly.
    - Local Dev: Must be able to fully test all code. Meaning there must be scaffolding for any backing service to run *locally*. If you need to connect to a VPN and external database to simply execute your local code than you don't have local dev environments and all deployments ought be halted until you do.

#####  Environment Summary
    The above environments, their management process, and validation of their adherance to the above conditions ought all be true at all times. Any deviation is huge total show stopping situation. I've never before worked with a software team wherein the previous sentence even needed be said. It's a constant truism across the field - and enforced, generally, from leadership. I get the impression leadership here is oppossed to even this basic, nearly rudamentary, level of rigor and care.

#### data schemas
Wouldn't normaly put this in definitions but I've never before asked, "Where are are migrations?"
and recieved the blank stares I get here. So maybe they need be defined a bit.

Dev's generally should not need access to external databases of any kind. There should just be a repo or small set of repos (generally shown on a dev's first day) that contain all the schema files and their migrations ( append only; basically CREATE and ALTER table statements ). If a dev needs to know how to write a query they simply look to the schema files. To verify and run tests they run those migrations in their local dev env's backing service. This is so ubiquitous there are many libraries that provide the setup. Go alone proves a handful (goose, migrate, atlas). Many folks use Alembic (python, from django iirc). Most orgs just roll their own as it's a simple problem and has more value in matching an org's particulars than in being general.  Why is any dev logging into a remote database at all?  I can imagine some usefulness of having direct DB login available in a developement environment. But no circumstance I've seen validates access to any higer env's db other than during an actual incident :: then only a small handful of dev's would have such access and they would rarely, if ever, use it. 

### Code 
 I should only need ask one question, get one answer, and then have more than enough work to keep me busy for a long time. That question, "where are your repos?" That's it. This is the first position, in my entire life, that I couldn't just look at your repos and know what to do. There are few contibutors to this sad state of affairs.

    1)  The code itself is wildly deviant from standard practices and conventions. This makes it very hard to read. I'm wanting to hold to positivity here so the code in this repo serves as an affirmative example of how to write Go. There are other good ways, and this was built very quickly so it likely has solid errors, but it is professional looking and has good test coverage. Further uses golang constructs in their intended manor and does not confound things by misusing conventions like `context`. 

    Here at OC, code intended as a broadly used library is getting integrated to production processes yet the library code is definitionally pre-alpha. Doesn't have tests - at all. Doesn't have a specification. Uses reflection when uncessary -- this maybe needs be emphasized -- In my experience, for any language that has reflection, if there is a way to do what you're doing w/o reflection than that is the end of the discussion; don't use reflection or explain why it is strictly necessary. Has persistence and biz logic code intermingled. Breaks many go conventions rendering it nearly unreadable. I've built (now with cueball) 4 workflow systems in my career. That I cannot read the code that implements such a system and cannot make heads or tails of why things are they way they are is concerning. This is engineering: If a line of code can be pointed to and the developer cannot say, affirmatively, why that line is there - that's an error and an issue that needs addressing. For any central library that rigor must be much more tightly adhered.

    2) No reqs, issues, designs, or specs of any note. When they do exist they appear created
    *after* the development process. Any developer with more than 5 years under their belt ought easily *without talking to anyone* be able to determine what any given repo does, how it connects to other repos, and how it deviates from what it ought to be ( the set of work still to be done ).
    3) PRs and _dev repos. The "_dev" repo thing is simply needless retardation. That's the soft, kind, understated way to state it. Never seen such sillyness -- just lock down master ( or main if you're gay ) and do regular PR's for merge to master like every single org in the field has been doing for over 2 decades.  *All* code discussion should happen in writing with PR's in version control. In general, in engineering overall, if it's not written down - in one utterly unambiguous place - then it doesn't exist. In software it's: if it isn't written down in version control then it doesn't exist. A big issue here is we use .doc .xls files for communication! Uh, maybe we could use that amazing technological inovation created decades ago ... Version control. Wow. Who would have thought of that. Oh, right, ever developer group ever. Why would any document getting edited by multiple people ever use a binary format for that document? Masochism is the only sane answer I can think of.

### developer communication
Stop it. Just stop it now. I've spoken to Paul, last week, on the phone more than any other person I know ever. I've never, in my life, had a developer call me to show me code. It's so dumb. make a branch and send it over. We can talk -- in writing, with version control - in a file within the branch or via a PR page. It's a huge waste of time but is downstream of a nastier patern here. That pattern is a kind of engineering suicide by a thousand self inflicted cuts (are we edgy dumb teanagers with a razor? No; we are something much more cring worthy.) Think I'll need an analogy to explain as our process is singular in the engineering world. If our development process were used to build a table we'd get 4 carpenters together have them talk talk talk then one will route the legs, another will cut the joinery for legs and surface, we'll then take turns putting the legs on while another dev planes the surface. When built we will then learn that the table need be adjusted on some dimension. Until OC don't think I've had a core task that was shorter than 12 weeks. Here, the constant tiny steps each one updated and enterred in xls, jira, teams, email, calls etc. Why all this extraneous sillyness? Atlassian is for business people not for devs, do you not know that?  When devs use it, it is generally as an opaque target -- meaning a githook might update a jira ticket from a dev's work, but the dev isn't logging into Jira -- it's not for them. Ops often use jira, idk why.
Each task I've been involved with here as had many many hands on it, yet each of these tasks has been smaller than what are generally given to a single person -- *AND* they have taken more total time. WTF.

### design, testing, and work allocation 
Our systems appear inverted. I spoke a bit on our last interview day about liking data
services: a centralized piece of software that serves up the datas. Mentioned, additionally,
good if on the wire. I did not, at that time, mean anything like what our data services are. 
The intention of a data service is to provide a single *conceptual* interface to your
business level data types. Essentially following the common HTTP pattern of 
`resources` and `collections`. If on the wire; simple request -> reply.
args:

 - for resources, id and sometimes id and type.
	returns only a single item (although it may have nested data)
 - for collections, various matches on contextually relevant fields
	returs a list and/or set of resources. sometimes just the ids

Callers of a data service should be entirely blind to if the data
is coming from a database, file, external integration, hyperborea, ... 
This layer facilitates -- actually more than that; makes possible -- sensible caching.

most of the software in any professional system should be blind to the details of 
persistence.  I went to lengths to demonstrate such in this cueball library. Not great lengths,
the interface and systems would be the same, but if not for demonstration I'd likly build only one one implementation for the Pipe, Record, and Blob interfaces.

#### testing
Basically this deserves a long bit but it's going to be simple.
OC must test. It currently does not. Running a handful of records through a process manually 
is not a test. How many simulataneous workers can resilient handle? What happens if I queue up
100k records in the postinfusion or naven input tables; or as you call them queue's - no one uses a db table as a queue lol. How about 1Mil? What if it's 1Mil and 50% are gonna error? We're not even addressing data issues (i.e. what if we url encode db string that's already url encoded?) Cueball stands as a clear example of the standard "how" here.
There, without exception, should be automated tests for all our code that must pass w/o manual intervention before deploy to staging or higher envs is possible at all. Any branch kicked off for artifact build will not create an artifact if there are any test failures and those tests must be comprehensive -- meaning must run `-race` tests for golang ( meaning you need a C compilier something else I've never been w/o ). Those tests must have good coverage and run _a lot_ of records ideally hitting avarious data edge cases. A lot means well over a thousand. Any process that doesn't have this setup doesn't belong in staging certainly not higher envs.


### Code Org and ownership ( authority and responsibility )
#### assignment of repos
#### relationship with business
#### code reviews


# imagine the `qt` situation.
## network (squad)
### get-boot-files
### gen-bridge
### dnsmasq | bind | coredns
### lighttp | nginx | caddy
## instance (platoon)
### *get-tap
### gen-vm







