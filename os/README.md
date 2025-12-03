# μCAPP Minimal OS

- Installed on depot Disk ID 0, Ring 0 (boot ring)
- Starts a 'shell' on the Tape device.
- Ring 0x00 on any drum is the drum boot program.
- Ring 0xff on any drum is the ring directory.

## CLI Commands

A `<title>` is a 4-letter name (6-bit ASCII) for the drum.
A `<drum>` is a 6-letter hexadecimal number for the drum.
A `<name>` is a 4-letter name for a ring.

```
DRUM                   ; Shows current drum status
BIND <title> <drum>    ; Connect a drum to the session
LIST <title>           ; Shows drum ring names
DROP <title>           ; Disconnect a drum from the session
TEST <title>[.<name>]  ; Test that a drum [or ring] can run.
EDIT <title>.<name>    ; Edit & debug a ring.
SPIN <title>           ; Spin the drum.
STOP <title>           ; Stop a drum.
COPY <src.name> <dst.name> ; Copy ring to drum, if space is available.
DELE <title.name>      ; Delete ring from drum index.
ALLO <title.name>      ; Allocate a ring, and name it.
EXIT                   ; Leave session.
```

### DRUM

```
APMOS) DRUM
Index        Title   State    Status
$00_0000     OS      7/7    SPIN     WATTS 47.299
$00_0001     TUT1    6/6    SPIN     WATTS 121.831
$F8_BD1E     JANI    1/8    UNKN     RING CORRUPTION DETECTED
$17_C003     ADMN    10/10  GOOD     DRUM MISS: FNAN, SECU
$00_0001     HR      6/6    SPIN     WATTS 121,831
$E0_7139     FNAN    10/10  FAIL     RING MISS: HR.EMPL
$91_2222     WARE    10/10  GOOD     READY
Current power usage is 290.961 watts, out of 500.000 watts available.
```

### LIST

```
APMOS) LIST OS
Title Ring  Status
DRUM  1     WATTS 0.001
BIND  2     WATTS 0.001
LIST  3     WATTS 0.001
DROP  4     WATTS 0.001
TEST  5     WATTS 0.001
EDIT  6     WATTS 0.001
SPIN  7     WATTS 0.001
STOP  8     WATTS 0.001
EXIT  9     WATTS 0.001
COPY  10    WATTS 0.001
DELE  12    WATTS 0.001
????  126   CORRUPTED
CLI   127   WATTS 10.281
```

```
APMOS) LIST FNAN
Title  Ring  Status
PAYR   12    CORRUPTED
INCM   22    WATTS 7.291
EXPN   77    DRUM MISS: WARE, POWR
COOK   127   WATTS 18.213
Predicted power use is 25.504
```

### BIND/DROP

Connect a drum to the session as a title.

```
APMOS) BIND FNAN $BEE0_7139
OK: Drum $BEE0_7193 attached to title FNAN.
APMOS) BIND FVAN $BEE0_7139
ERROR: Drum $BEE0_7139 already attached as FNAN.
APMOS) BIND FNAN $0000_0002
ERROR: Title FNAN already bound, use `DROP FNAN` first to remap.
APMOS) DROP OS
ERROR: OS drum cannot be dropped at this time.
```

### TEST

Test a drum or a ring on the drum.

```
APMOS) TEST HR
Marking HR as UNKN
TEST HR.EMPL: PASS
TEST HR.PAYR: PASS
TEST HR.HIRE: PASS
TEST HR.EXIT: PASS
OK: All tests pass, HR marked as GOOD
APMOS) TEST FINA.CIAL
ERROR: No such drum/ring as FINA.CIAL
APMOS) TEST FNAN.CIAL
Marking FNAN as UNKN
TEST FNAN.CIAL: FAIL
ERROR: Test(s) failed, FNAN marked as FAIL
```

### SPIN/STOP

Spin/stop a drum.

```
APMOS) SPIN HR
Spinning up HR...
OK: WATTS +45.123
APMOS) SPIN HR
OK: WATTS +0.000
APMOS) SPIN ADMN
ERROR: ADMN is not GOOD.
APMOS) SPIN FNAN
Spinning up FNAN...
ERROR: CIRCUIT BREAKER TRIPPED, TOTAL WATTS 671.212 > 500.000
APMOS) STOP HR
Stopping HR...
OK: WATTS -45.123
APMOS) SPIN FNAN
OK: WATTS +134.882
```

### EDIT

Edit a ring

```
APMOS) EDIT OS.EDIT
ERROR: OS drum is spinning, and cannot be edited.
APMOS) EDIT HR.EMPL
ERROR: HR drum is spinning, and cannot be edited.
APMOS) STOP HR.EMPL
Stopping HR...
APMOS) EDIT HR.EMPL
Starting editor...
...
OK: Untested changes saved to HR.EMPL, HR changed from GOOD to UNKN
ERROR: Failed changes saved to HR.EMPL, HR changed from GOOD to FAIL
OK: Tested changes saved to HR.EMPL, HR is GOOD
APMOS) EDIT HR.ZZZZ
ERROR: No such ring HR.ZZZZ
```
