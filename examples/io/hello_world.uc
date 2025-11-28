; Load the string 'Hello World' into the tape

list of ARENA_FREE ARENA_MASK
list all
write first 'H'
list next
write first 'e'
list next
write first 'l'
list next
write first 'l'
list next
write first 'o'
list next
write first ' '
list next
write first 'W'
list next
write first 'o'
list next
write first 'r'
list next
write first 'l'
list next
write first 'd'
list next
write first '!'
list next
write first '\n'
list next
list not
store tape 0xff
list not
write list ~0 ~0
