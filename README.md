# Forum

Forum sections and topics for [Kosatka](https://github.com/kosatka-cms/kosatka): the `/forum` pages - sections from a vocabulary, topics as content of their own type with threaded replies, the active and unanswered topics - and the forum settings.

The **Forum** module (`forum`) gives a site a classic discussion forum:
categories with sections, topics in the sections and threaded replies
under each topic. The forum's structure is a taxonomy vocabulary that
you choose, its topics are content of the module's own *Forum* type, and
the replies are ordinary comments.

## Install

A site with modules from outside the engine builds its own binary
(`kosatka.Main`); `app module add` adds this one to it:

```bash
app module add github.com/kosatka-cms/kosatka-module-forum
go build -o app .
```

Then enable *Forum* on the site's Modules page: it installs its
tables and starter rows. See [Using modules](https://kosatka-cms.github.io/kosatka/modules/using-modules/).

A new site built with it gets it with `app install -modules forum,…`
(or `-modules all`).

A site installed with the forum and `-demo` has it set up: a *Forum*
vocabulary with two categories and their sections, a few topics with
replies by the demo users, and a *Forum* link in the main menu.

## What it adds

| What | Where |
|---|---|
| The forum's front page: every category with its sections | `/forum` |
| A category's page: the same table for one category | the category term's address |
| A section's page: its topics | the section term's address, for example `/forum/questions` |
| *Active topics*: topics of all sections by their latest post | `/forum/active` |
| *Unanswered topics*: topics without replies, newest first | `/forum/unanswered` |
| A topic's page with its replies | `/forum/<topic-slug>` |
| *Create new topic* | `/forum/new-topic/<section address>` |
| *Edit topic*, *Delete topic* | `/forum/topic/<id>/edit`, `/forum/topic/<id>/delete` |
| The settings page | **Configuration → Modules → Forum** (`/admin/config/forum`) |
| The *Forum* content type (forum topics) | its permissions are on **People → Permissions** |

The section pages and the topics are in the sitemap. `robots.txt` keeps
search engines off the create, edit and delete pages, and the *Active
topics* and *Unanswered topics* lists are marked not to be indexed.

The module has no blocks of its own. The topic page shows the blocks of
the *Inside content (content page)* region (`content_inline`) between
the topic and its replies, as content pages do (see
[Blocks](https://kosatka-cms.github.io/kosatka/guides/blocks/)).

## How the forum is built

The forum reads its structure from one vocabulary (see
[Taxonomy](https://kosatka-cms.github.io/kosatka/guides/taxonomy/)):

- a **root term** is a **category**. It only groups sections: its page
  lists them, and it holds no topics;
- a **child term** of a category is a **section**: the place where topics
  are posted;
- deeper terms are not shown on the forum's pages.

Categories and sections come in the order of their weight, then name.
A term's name and description are shown in the tables; the term's
address is the address of its forum page.

Until a vocabulary is chosen, the forum is empty.

## Set up the forum

1. Open **Taxonomy** (`/admin/config/taxonomy`) and create a vocabulary
   for the forum only, for example "Forum", with **Term path prefix**
   `forum`. Leave **Term pages** off: the forum shows these terms itself.
2. Add the categories as top-level terms, for example "General" and
   "Off topic".
3. Add the sections as their children, for example "Questions" and
   "Announcements" under "General". With an empty **Path**, a term gets
   the vocabulary prefix plus its name (`forum/questions`), which is
   where its forum page will be.
4. Open **Configuration → Modules → Forum** (`/admin/config/forum`),
   choose the vocabulary in **Forum vocabulary** and click **Save**.
5. Check who may post: installing the module gives the *Signed-in
   user* role *Create* in the **Forum** group of **People →
   Permissions**, so every signed-in user can start topics. Untick it
   there, or give it to other roles, to choose who may.
6. Add a link to `/forum` to a menu (see [Menus](https://kosatka-cms.github.io/kosatka/guides/menus/)).

The prefix doesn't have to be `forum`: a section whose address starts
with the vocabulary's prefix (say `boards/general/questions`) opens at
that address too. Keep the terms' addresses under the prefix, or the
links on the forum pages won't open.

To change sections later, edit the terms: renaming, reordering or adding
a section shows on the forum at once. To switch the forum off without
disabling the module, choose *- not selected (forum disabled) -* and
save.

## Post a topic

A user with the *Create* permission of the **Forum** type sees *Create
new topic* above and below a section's topics; a guest sees *Log in to
post new forum topics.* instead.

The form has:

| Field | |
|---|---|
| Forum section | the section; the current one is selected. The categories are listed as headings and can't be chosen. Required |
| Title | required, up to 255 characters |
| Body | the text, in a visual editor with headings, lists, links and images from the user's files |

**Create topic** publishes the topic at once and opens it. Its address
is `/forum/` plus the title, transliterated (`-1`, `-2`… is added when
it's taken). Editing the title later doesn't change the address.

## Replies

The replies are the topic's comments. A signed-in user presses *Reply*
(under the topic or under a reply, to answer that reply); a guest sees
*Log in or register to post comments*. Replies come 50 per page; *Last
post* jumps to the newest one.

The comment rules are the site's usual ones: a reply goes through the
site's comment checks (see
[Comments](https://kosatka-cms.github.io/kosatka/guides/comments/#what-happens-to-a-new-comment)), so
it is published at once or waits for approval - by default, a new
author's first one waits. Authors edit their own replies, and roles
with *Administer comments* edit and delete any. Forum replies are kept
out of the site's *Recent comments* block.

When the *Private messages* module is on, other signed-in users see a
link to send the topic's author a message next to the author's name.

## Moderate topics

On a topic's page, a user who may do so sees these buttons:

| Button | Who sees it |
|---|---|
| **Pin to top** / **Unpin** | roles with *Edit any* of the **Forum** type. A pinned topic leads its section's list, marked *Sticky:* |
| **Edit** | roles with *Edit any*, and the author with *Edit own* while the topic is published |
| **Delete** | roles with *Delete any* |

**Edit** opens the same form as creating. Changing **Forum section**
moves the topic to another section.

**Delete** asks for confirmation and removes the topic with all its
replies for good, along with the files nothing else uses.

Forum topics also appear in the admin's content list (**Content**),
where they can be published, unpublished or deleted in bulk like other
content. An unpublished topic disappears from the forum's lists and
counts.

With the *Action log* module on, its *Forum* group records a topic moved
to another section (from which to which), pinned and unpinned.

## Topic lists

A section's page lists 25 topics per page: pinned ones first, then by
the latest post. Click a column heading to sort by it, and again to
reverse:

| Column | |
|---|---|
| Topic / Topic starter | the title, the author and the date |
| Replies | the number of replies; a topic with more than 20 is shown as a *Hot topic* |
| Views | how many times the topic was viewed |
| Last post | who replied last and when |

*Active topics* and *Unanswered topics* use the same table, across all
sections, with each topic's section under its title.

## Statistics

Below the categories, the front page shows *Statistics*: the number of
published topics, of replies (*Posts*) and of the site's users, and
*Newest member:* with the five newest active accounts (not blocked, not
awaiting approval). Each section row has its own *Topics* and *Posts*
counts and its *Last post*: the topic with the latest activity.

## Disable and uninstall

While the module is disabled, the `/forum` pages answer 404 and the
topics don't open; the topics, replies and settings stay.

Uninstalling (see [Using modules](https://kosatka-cms.github.io/kosatka/modules/using-modules/#uninstall-a-module))
first deletes the forum topics with their replies, then the module's
tables and settings. The forum's vocabulary lives in the core's taxonomy:
uninstalling deletes it with its terms too, unless something else uses
it (a taxonomy field of a content type, a search setting, or content
tagged with its terms). In that case it stays, and you can delete it
yourself on the **Taxonomy** page.

## Development

Its strings are translated in `translations/<lang>.json`.

The module is built on the engine's module SDK only
(`github.com/kosatka-cms/kosatka/modulekit`, `pkg/*`).

```bash
go test ./...
# with MySQL and PostgreSQL (the engine's docker-compose.test.yml):
KOSATKA_TEST_MYSQL_DSN="root:test@tcp(127.0.0.1:33306)/?parseTime=true" \
KOSATKA_TEST_POSTGRES_DSN="postgres://postgres:test@127.0.0.1:35432/postgres?sslmode=disable" \
  go test -tags dbintegration ./...
```

A schema change: edit `ent/schema`, `go generate ./ent`, then
`go run ./cmd/migrations -name <name>` (both test databases running) -
it writes the next migration for MySQL and PostgreSQL into
`ent/migrations`; `go run ./cmd/migrations -check` fails while the files
are behind the schema. See [Write a module](https://kosatka-cms.github.io/kosatka/modules/write-a-module/).

## License

MIT
