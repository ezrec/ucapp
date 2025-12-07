# μCAPP Command Line Tool

## Compile

`ucapp build somefile.uc`

## Save a file to a drum by name.

`ucapp depot save --drum 0x123456 QUX somefile.ur`

## Delete a file by name on a drum

`ucapp depot delete --drum 0x123456 QUX`

## List all files in the depot

`ucapp depot list`

## List all files on a specific drum

`ucapp depot list --drum 0x123456`

## Save a file to a drum directly

`ucapp depot save --drum 0x123456 0xAB somefile.ur`

## Execute a drum in the depot

`ucapp run --drum 0x123456`

## Execute a ring on a drum in the depot

`ucapp run --drum 0x123456 --ring 0xAB`

## Execute a drum with a specific input and output tape

`ucapp run --drum 0x123456 --input <in.tape> --output <out.tape>`

