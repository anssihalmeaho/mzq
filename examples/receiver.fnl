
ns main

import mzqbro
import mzqque

main = proc()
	options = map(
		'own-name' 'receiver'
		'own-addr' ':8082'
		'addrs' list('127.0.0.1:8081')
	)
	_ _ broker = call(mzqbro.new-broker options):
	my-queue = call(mzqque.new-queue 5)

	# spawn queue listener
	_ = spawn(call(proc()
		_ = print('received:' call(mzqque.getq my-queue))
		while(true 'none')
	end))

	# register queue
	_ = print('reg: ' call(mzqbro.reg-queue broker 'some-queue' my-queue))

	# spawn own sender
	_ = spawn(call(proc()
		import stdtime
		_ = call(stdtime.sleep 4)
		_ = print('send local: ' call(mzqbro.send-msg broker 'receiver' 'some-queue' map('Hello' 'World')))
		while(true 'none')
	end))

	# waiting loop
	call(proc()
		import stdtime
		_ = call(stdtime.sleep 2)
		_ = print('...')

		while(true 'none')
	end)
end

endns

