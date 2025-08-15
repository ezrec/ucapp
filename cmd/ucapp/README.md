# μCAPP Command Line Tool

## Compile and Run

`ucapp -c <somefile.uc>`

## Compile and save to a Ring

`ucapp -c <somefile.uc> -s -r <somefile.ring>`

## Load a ring into the CAPP and execute

`ucapp -r <somefile.ring>`

## Query a ring's attributes

`ucapp -r <somefile.ring>` -q

## Add a ring to a drum

`ucapp -d <somefile.drum> -a <index> -r <somefile.ring>

NOTE: <somefile.drum> must have a format compatible to the <somefile.ring>

## Execute a drum

`ucapp -d <somefile.drum>`

## Query a drum's attributes

`ucapp -d <somefile.drum>` -q

## Execute a drum at a specific ring index

`ucapp -d <somefile.drum> -a <index>`

## Execute a drum with an input and output tape

`ucapp -d <somefile.drum> -i <in.tape> -o <out.tape>`

