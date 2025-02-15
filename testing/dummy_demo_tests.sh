#!/usr/bin/env bash

set -ex

FOLDER_SRC="/tmp/src-$RANDOM"
FOLDER_DEST="/tmp/dest-$RANDOM"

# Create directories
mkdir -p "$FOLDER_SRC/snes"
mkdir -p "$FOLDER_SRC/nes"
mkdir -p "$FOLDER_SRC/psx/multidisk"
mkdir -p "$FOLDER_SRC/psx/images"
mkdir -p "$FOLDER_SRC/atari2600"
mkdir -p "$FOLDER_SRC/gba"
# Create SNES game files
touch "$FOLDER_SRC/snes/chrono_legends.sfc"
touch "$FOLDER_SRC/snes/mega_mech_warriors.smc"
# Create NES game files
touch "$FOLDER_SRC/nes/castle_crawlers.nes"
touch "$FOLDER_SRC/nes/robo_samurai.nes"
touch "$FOLDER_SRC/nes/galaxy_voyager.nes"
# Create PSX game files
touch "$FOLDER_SRC/psx/final_fantasy_viii.bin"
touch "$FOLDER_SRC/psx/multidisk/xenogears_disk1.bin"
touch "$FOLDER_SRC/psx/multidisk/xenogears_disk2.bin"
touch "$FOLDER_SRC/psx/images/final_fantasy_viii.png"
touch "$FOLDER_SRC/psx/images/xenogears.png"
# Create Atari 2600 game files
touch "$FOLDER_SRC/atari2600/cosmic_crusader.a26"
touch "$FOLDER_SRC/atari2600/pixel_pirates.a26"
touch "$FOLDER_SRC/atari2600/neon_nights.a26"
# Create Game Boy Advance game files
touch "$FOLDER_SRC/gba/dragon_quest_legends_romhack.gba"
touch "$FOLDER_SRC/gba/cyber_ninja_revolution.gba"
# Create files with content
echo "<xml>SNES Game Data</xml>" > "$FOLDER_SRC/snes/gamelist.xml"
echo -e "./multidisk/xenogears_disk1.bin\n./multidisk/xenogears_disk2.bin" > "$FOLDER_SRC/psx/multidisk/xenogears.m3u"
# Create PSX gamelist
cat << EOF > "$FOLDER_SRC/psx/gamelist.xml"
<gameList>
  <game>
    <path>./final_fantasy_viii.bin</path>
    <name>Final Fantasy VIII</name>
    <desc>Embark on an epic journey with Squall and his companions in this reimagined classic RPG.</desc>
    <image>./images/ff8_cover.png</image>
  </game>
  <game>
    <path>./metal_gear_solid_2.bin</path>
    <name>Metal Gear Solid 2: Shadows of Liberty</name>
    <desc>Experience the next chapter in Solid Snake's saga in this thrilling stealth action game.</desc>
    <image>./images/mgs2_cover.png</image>
  </game>
</gameList>
EOF

# Create Atari 2600 gamelist
cat << EOF > "$FOLDER_SRC/atari2600/gamelist.xml"
<gameList>
  <game>
    <path>./cosmic_crusader.a26</path>
    <name>Cosmic Crusader</name>
    <desc>Defend Earth from alien invaders in this action-packed space shooter!</desc>
  </game>
  <game>
    <path>./pixel_pirates.a26</path>
    <name>Pixel Pirates</name>
    <desc>Sail the 8-bit seas, plunder treasure, and become the most feared pirate in the pixelated world!</desc>
  </game>
  <game>
    <path>./neon_nights.a26</path>
    <name>Neon Nights</name>
    <desc>Race through a retro-futuristic city in this high-speed neon-drenched adventure!</desc>
  </game>
</gameList>
EOF

# Create Game Boy Advance gamelist
cat << EOF > "$FOLDER_SRC/gba/gamelist.xml"
<gameList>
  <game>
    <path>./dragon_quest_legends_romhack.gba</path>
    <name>Dragon Quest Legends</name>
    <desc>Embark on an epic journey to save the kingdom from an ancient dragon in this classic RPG!</desc>
  </game>
  <game>
    <path>./cyber_ninja_revolution.gba</path>
    <name>Cyber Ninja Revolution</name>
    <desc>Master the arts of stealth and technology as a futuristic ninja in this action-packed platformer!</desc>
  </game>
</gameList>
EOF

mkdir -p "$FOLDER_DEST/SFC"
mkdir -p "$FOLDER_DEST/FC"
mkdir -p "$FOLDER_DEST/PLAYSTATION"
mkdir -p "$FOLDER_DEST/2600"
mkdir -p "$FOLDER_DEST/GAMEBOYADVANCE"

set +ex

echo "$FOLDER_SRC created"
echo "$FOLDER_DEST created"

echo "Basic copy invocation"
cat << EOF
go run ROMCopyEngine.go \\
    --sourceDir $FOLDER_SRC \\
    --targetDir $FOLDER_DEST \\
    --mapping psx:PLAYSTATION \\
    --mapping snes:SFC \\
    --mapping nes:FC \\
    --mapping gba:GAMEBOYADVANCE \\
    --mapping atari2600:2600 \\
    --copyExclude '**/*_romhack*' \\
    --explodeDir multidisk \\
    --explodeDir images \\
    --rename gamelist.xml:data.xml \\
    --rewritesAreRegex \\
    --rewrite "*.m3u:\./multidisk:./" \\
    --rewrite "*.xml:\.\./.*?/images:./" \\
    --skipConfirm --skipSummary
EOF

echo "Shorter:"

cat << EOF
go run ROMCopyEngine.go \\
    --sourceDir $FOLDER_SRC \\
    --targetDir $FOLDER_DEST \\
    --mapping psx:PLAYSTATION \\
    --mapping gba:GAMEBOYADVANCE \\
    --mapping atari2600:2600 \\
    --copyExclude '**/*_romhack*' \\
    --explodeDir multidisk \\
    --explodeDir images \\
    --rename gamelist.xml:data.xml \\
    --rewritesAreRegex \\
    --rewrite "*.m3u:\./multidisk:./" \\
    --rewrite "*.xml:\.\./.*?/images:./" \\
    --skipConfirm --skipSummary
EOF
