# Standard findings

Some findings are marked standard. A standard review ran on the same change beside the reviews. It checks fixed rules by the form of each statement, once for every database, and it posts its comment whatever you answer. A standard finding has no database, its title names its rule, and its evidence is the comment it posts.

- Put a finding of a review in a list with a standard finding when it reports the operation the standard finding flags: the same condition, or something that follows from that operation, such as an object that breaks because of it. The author then reads one comment about the operation, with everything that follows from it.
- A finding that reports the very condition of the standard finding always goes with it.
- A finding about another operation of the statement, or about a problem that the standard finding's operation does not cause, stays out of its list.
- The rules above still decide which findings of the reviews are the same problem. Different problems on one operation are different lists, and each of them holds the standard finding. So a standard finding can be in more than one list.
- A list holds at most one standard finding, and never a standard finding alone. A standard finding with no finding of a review on its operation is in no list.
- An earlier finding goes by the same rules. It can share a list with a standard finding.

# Comments already posted

A standard finding that names a thread already has a comment, from an earlier review of the change. The comment holds the text of the thread's standard findings, and what the findings of that earlier review added to it.

- Put the thread in `same` when its comment already states every problem that you put in a list with one of its standard findings, and states no other problem. Its comment then stays as it is.
- Wording, numbers, the order of the problems, and the databases named do not make a difference. A problem that the comment lacks does, and so does a problem it states that no finding of this review reports.
- The comment can also state facts about the statement without calling them a problem, such as how much data it touches or what else uses an object. A fact that no finding reports makes no difference. A problem is stated when the comment states its effect, even as a fact and with no fix.
- When you are unsure, leave the thread out of `same`. Its comment is then written again from the findings of this review.

# Answer format with standard findings

{"groups": [[9, 1, 4], [9, 2], [3, 5]], "same": [1]}

- groups: as above, except that the id of a standard finding can be in more than one list, or in none.
- same: the ids of the threads whose comment stays. Leave the list empty when there is none.
