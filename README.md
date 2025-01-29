# cueball
Basic workflow setup. Initially built to demonstrate golang design patterns.

# Motivating circumstances
This project is largely motivated tangentially; by circumstances seen in the field
that have cost great quantities of time and money. 

## Specific situation 
Nearly every fundamental process here is different than I've seen elsewhere. Many 
are directly at odds with what I've seen. Despite the fact other orgs have had
staggering amounts of waste and foolishness; the worst of them is like a wee little
candle to the sun of absurdity shown brightly by the methodologies in place at OC.

I want to underscore as boldly as possible that the circumstances below have 
been effectively ubiquitous; OC is a complete outlier and utterly unique in my 20 years.

### deployment environments
#### basic definitions

    - Production: Artifact only deployments. Isloated: any manual interaction *at all* is an "incident" or is becuase of an "incident". No exceptions at all. If you need manual interaction to deploy software than you, definitionally, do not have a production system and all deployments ought be hard halted until you do. Can be ephemerally deployed using disaster recovery data for population. Previous releases (rollbacks) are completely reliable and evidenced - but not necessarily easy b/c of the data.
    - UAT: Artifact only deployments. Isoloated: no manual interaction but no incident if there is provided UAT is ultimatlely still entirely created by automation. UAT is identical to production on moment of prod deploy and may be one version ahead for any deployable at other times. Can be deployed ephemerally (automatically: push button) other than the actual data itself. An new UAT deployment is, definitionally, a staging environment. UAT has "real" data although it may be obfuscated for PII reasons. That data difference and the size of the environment (number of machines) are the only allowed constant deviations from PROD. rollbacks push button and reliable -- should be easy.
    - Staging: Artifact only deployments. Manual interaction allowed, but when manual interaction occurs it ought trigger a full redploy of staging before next prod release. Can have all kinds of mixmatched versions but generally is like a UAT that is smaller and devs/QA/ops will broadly have access to the data and backing services.  Putting it in a broken state is allowed if everyone involved is aware. Generally, however, if an action risks breakage then you ought simply deploy a new staging and work there. Can deploy an previous release, but not done as rollback -- just a new deploy with older software versions. -- easy because no data reqs other than schema and all artifact deploy.
    - Development: Can be deployed to directly by engineering staff. Is easily and individually ephemeral. Meaning *any* dev can say (ideally to an RPC or REST endpoint) "give me a new staging" and get one quickly by an automated process. Additionally such environments can have any versions for any deployed components.  The initial deployment of a dev environment will be pushbutton and artifact only; but further updates can be willy nilly.
    - Local Dev: Must be able to fully test all code. Meaning there must be scaffolding for any backing service to run *locally*. If you need to connect to a VPN and external database to simply execute your local code than you don't have local dev environments and all deployments ought be halted until you do.


#### data schemas
Dev's generally should not need access to external databases of any kind. There should just be a repo or small set of repos (generally shown on a dev's first day) that contain all the schema files and their migrations ( append only after concat; CREATE and ALTER table statements ). If a dev needs to know how to write a query they simply look to the schema files. To verify they run those migration in their local dev env's backing services. 


####  summary
    The above environments, their management process, and validation of their adherance to the above conditions ought all be true at all times. Any deviation is huge total show stopping situation. I've never before worked with a software team wherein the previous sentence even needed be said. It's a constant truism across the field - and enforced, generally, from leadership. I get the impression leadership here is oppossed to even this basic, nearly rudamentary, level of rigor and care.

### code change process
 I should only need ask one question, get one answer, and then have more than enough work to keep me busy for a long time. That question, "where are your repos?" That's it. This is the first position, in my entire life, that I couldn't just look at your repos and know what to do. There are few contibutors to this sad state of affairs.

    1)  The code itself is wildly deviant from standard practices and conventions. This makes it very hard to read. The code in this repo serves as an example. Code intended as a broadly used library is getting integrated to production processes yet the library code is pre-alpha. Doesn't have tests - at all. Has persistence and biz logic code intermingled. Breaks many go conventions rendering it nearly unreadable. I've built (now with cueball) 4 workflow systems in my careers. That I cannot read the code that implements such a system and cannot make heads or tails of it sucks ass.
    2) No reqs, issues, designs, or specs of any note. When they do exist they appear created
    *after* the development process. Any developer with more than 5 years under their belt ought easily *without talking to anyone* be able to determine what any given repo does, how it connects to other repos, and how it deviates from what it ought to be ( the set of work still to be done ).
    3) PRs and _dev repos. The "_dev" repo thing is simply needless retardation. That's the soft, kind, understated way to state it. Never seen such sillyness -- just lock down master ( or main if you're gay ) and do regular PR's for merge to master like every single org in the field has been doing for over 2 decades.  *All* code discussion should happen in writing with PR's. In general, in engineering overall, if it's not written down - in one unambiguous place - then it doesn't exist. More on this in a later section - but a big issue here is we use .doc .xls files for communication! Uh, maybe we could use that amazing technological inovation created decades ago ... Version control. Wow. Who would have thought of that. Oh, right, ever developer group ever. Why would any document getting edited by multiple people every use a binary format for that document? Masochism is the only sane answer there.

### developer communication
Stop it. Just stop it now. I've spoken to Paul, last week, on the phone more than any other person I know ever. I've never, in my life, had a developer call me to show me code ever. It's so dumb. make a branch and send it over. We can talk -- in writing, with version control - in a file within the branch or via a PR page. It's a huge waste of time but is downstream of a naster patern here. That pattern is a kind of engineering suicide by a thousand self inflicted cuts (are we edgy dumb teanagers with a razor? No; we are something much more cring worthy.) Think I'll need an analogy to explain as our process is singular in the engineering world. If our development process were used to build a table we'd get 4 carpenters together have them talk talk talk then one will route the legs, another will cut the joinery for legs and surface, we'll then take turns putting the legs on while another dev planes the surface. When built we will then learn that the table need be adjusted on some dimension. Until OC don't think I've had a core task that was shorter than 12 weeks. The constant tiny steps each one updated and enterred in xls, jira, teams, email, calls etc. Why all this extraneous sillyness? Atlassian is for business people not for devs, do you not know that?  When devs use it, it is generally as an opaque target -- meaning a githook might update a jira ticket from a dev's work, but the dev isn't logging into Jira -- it's not for them. 

### design, testing, and work allocation 


# imagine the `qt` situation.
## network (squad)
### get-boot-files
### gen-bridge
### dnsmasq | bind | coredns
### lighttp | nginx | caddy
## instance (platoon)
### *get-tap
### gen-vm







