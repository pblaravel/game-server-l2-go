# Java datapack

Vendored from https://github.com/shnok/l2-unity-gameserver `gameserver/data/xml`.
Geodata, HTML dialogs, the disabled spawnlist and the Java quest script tree
are still not copied.

The gameserver parses every vendored XML file on start. Character create grants
**autoGet** skills only (`cost="0"` and `minLvl` ≤ level), same as Java, and
uses the first `<spawns>` point from the class XML.

Live systems that already consult the extra tables:

- armor set bonuses in `RecalcStats`
- spellbooks on skill learn
- heal SPS correction
- newbie guide `SupportMagic`
- instant teleports
- peace-zone lookup
- XML spawnlist NPCs next to the hardcoded Talking Island set
- `is_dropable` / `crystal_type` on items
