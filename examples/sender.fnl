
ns main

import mzqbro

main = proc()
	options = map(
		'own-name' 'sender'
		'own-addr' ':8081'
		'addrs' list('127.0.0.1:8082')
	)
	_ _ broker = call(mzqbro.new-broker options):

	# waiting loop
	call(proc()
		import stdtime
		_ = call(stdtime.sleep 1)
		_ = print('send: ' call(mzqbro.send-msg broker 'receiver' 'some-queue' list('Hello' 'There')))

		while(true 'none')
	end)
end

endns

