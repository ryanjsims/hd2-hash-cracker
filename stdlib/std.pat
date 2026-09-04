#{known = #{wordlist "known_hashes.txt"}}
#{known-words = #{split #{known} "/_:"}}
#{words-10kplus = #{wordlist "wordlist_10k_plus.txt"}}
#{strings-words = #{wordlist "strings_words.txt"}}