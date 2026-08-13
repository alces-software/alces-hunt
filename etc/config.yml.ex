---
# Host to send data to when running client (optional)
# target_host:
#
# Port to listen/send over. This must match on both server and client. (required)
# port:
#
# Mode to use when autorun (optional). Must be one of "hunt" or "send".
# autorun_mode:
#
# Automatically include self when hunting (optional)
# include_self: false
#
# Command to use to generate content (optional)
# content_command:
#
# Allow existing nodes when hunting/parsing (optional)
# allow_existing: false
#
# Key used to authenticate data (required for non-empty protection)
# auth_key: alces-hunt
#
# Broadcast address to use when sending via a UDP broadcast (optional)
# broadcast_address: 255.255.255.255
#
# Command used to run the profile executable (optional)
# Defaults to $ALCES_HUNT_ROOT/bin/alces-hunt profile
# profile_command:
#
# Automatically parse nodes when hunted if they match this regular expression
# auto_parse: /^cnode/
#
# Automatically trigger a profile identity when a parsed label matches.
# Keys are regular expressions; values are identity names.
# auto_apply:
#   exp1: identity1
#   exp2: identity2
#
# Preset data sent by the client (and by hunt --include-self)
# presets:
#   label:
#   prefix:
#   groups:
#     - example_group1
#     - example_group2
#
# How to process a node's hostname into a default label.
# Must be one of "long", "short" or "blank" (optional, default "long")
# default_label: long
#
# Default numeric start value for automatically parsed nodes
# default_start: "01"
#
# Custom start values for given prefixes
# prefix_starts:
#   cnode: "001"
#
# If automatic parsing would create a label that already exists,
# skip that index and keep incrementing until an unused label is found.
# skip_used_index: true
#
# If the server cannot be reached, retry every n seconds
# retry_interval: 5
