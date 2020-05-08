# 2048ML
This is my hardcoded, multithreaded, neural net implementation in go, which teaches itself how to play the game 2048 genetically.  
Actual 2048 game adaptation cred: https://github.com/andrewstuart/2048.go . 

Haven't written any saving function/used any library, so when main() returns, the trained neural network dies.

### How it works:
The first layer in the neural network has 16 nodes, each corresponding to every tile in the 4x4 game board  
of 2048. These 16 nodes are connected to a single hidden layer of 8 nodes, which then feed to the  
four different output nodes each representing the direction of a move (left, right, up or down).

The input nodes (16) have no biases and no incoming connections, they are the eyes of the network.
The middle "hidden" layer's (8) nodes have both biases and weights associated with their connections  
coming from every node in the input layer.
The output nodes (4) are identical to the "hidden" layer's (8) nodes, but connect to the hidden layer.

When playing the game, x neural networks are spawned and play fully randomly, the ones that succed the best  
are kept and reproduce to replace some (y / z) amount/ratio of the less succeding networks. This procedure attempts  
to mimic the force "survival of the fittest", and genetically drives the populace of networks to adopt  
playstyles that result in high game scores.
