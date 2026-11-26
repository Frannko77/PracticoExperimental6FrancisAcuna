package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// estudiante es el nombre que aparece en el reporte del día.
const estudiante = "Francis Acuña"

// Sucursal guarda el nombre de una farmacia, cuánto tarda en responder y sus ventas del día.
type Sucursal struct {
	Nombre string
	Demora time.Duration
	Ventas []float64
}

// sucursales son las cuatro farmacias de la cadena; Cumbayá no envió sus ventas de hoy.
var sucursales = []Sucursal{
	{"Centro", 400 * time.Millisecond, []float64{120.50, 89.90, 202.10}},
	{"Norte", 600 * time.Millisecond, []float64{150.00, 239.20}},
	{"Cumbayá", 800 * time.Millisecond, nil},
	{"Sur", 1000 * time.Millisecond, []float64{75.80, 200.00}},
}

// Consolidado reúne lo que reportan las sucursales; el candado mu protege sus datos.
type Consolidado struct {
	mu         sync.Mutex
	Total      float64
	Reportadas int
}

// Sumar agrega el total de una sucursal; el candado deja pasar a una sola goroutine a la vez.
func (c *Consolidado) Sumar(total float64) {
	c.mu.Lock()         // si otra goroutine está sumando, esta espera aquí
	defer c.mu.Unlock() // el candado se abre al salir de la función
	c.Total += total
	c.Reportadas++
}

// consultar pide las ventas a una sucursal y devuelve el total y la venta mayor.
func consultar(s Sucursal) (float64, float64) {
	time.Sleep(s.Demora) // simula el tiempo que tarda la sucursal en responder por la red
	mayor := s.Ventas[0] // si la sucursal no envió ventas, aquí ocurre un panic
	total := 0.0
	for _, v := range s.Ventas {
		total += v
		if v > mayor {
			mayor = v
		}
	}
	return total, mayor
}

// cierre consulta las sucursales a la vez y escribe el reporte del día en w (consola o página).
func cierre(w io.Writer) time.Duration {
	inicio := time.Now()
	var wg sync.WaitGroup
	var c Consolidado
	lineas := make([]string, len(sucursales)) // una casilla fija por sucursal para mantener el orden
	var secuencia time.Duration

	for i, s := range sucursales {
		secuencia += s.Demora
		wg.Add(1) // registra una tarea pendiente más
		go func(i int, s Sucursal) {
			defer wg.Done() // al terminar, avisa que esta sucursal ya respondió
			defer func() {  // si hay un panic, se recupera aquí y el servidor sigue funcionando
				if r := recover(); r != nil {
					lineas[i] = fmt.Sprintf("%s: sin datos - fallo recuperado: %v", s.Nombre, r)
				}
			}()

			total, mayor := consultar(s)
			c.Sumar(total)
			lineas[i] = fmt.Sprintf("%s: %d ventas, total $%.2f, venta mayor $%.2f", s.Nombre, len(s.Ventas), total, mayor)
		}(i, s)
	}

	wg.Wait() // espera a que respondan todas las sucursales
	duracion := time.Since(inicio)

	fmt.Fprintln(w, "Cierre del día - Farmacias del Valle")
	fmt.Fprintln(w, "Programado por:", estudiante)
	fmt.Fprintln(w)
	for _, l := range lineas {
		fmt.Fprintln(w, l)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Sucursales reportadas: %d de %d\n", c.Reportadas, len(sucursales))
	fmt.Fprintf(w, "Total del día: $%.2f\n", c.Total)
	fmt.Fprintf(w, "Tiempo de respuesta: %.1f s (en secuencia: %.1f s)\n", duracion.Seconds(), secuencia.Seconds())
	return duracion
}

// verCierre atiende la ruta /cierre: escribe el reporte en la página y anota cuánto tardó.
func verCierre(w http.ResponseWriter, r *http.Request) {
	duracion := cierre(w)
	fmt.Printf("Consulta atendida en %.1f s\n", duracion.Seconds())
}

// main publica el reporte en http://localhost:8080/cierre y espera solicitudes.
func main() {
	http.HandleFunc("/cierre", verCierre)
	fmt.Println("Servidor de Farmacias del Valle en http://localhost:8080/cierre")
	// ListenAndServe se queda atendiendo; cada solicitud corre en su propia goroutine
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("No se pudo iniciar el servidor:", err)
	}
}

// Mi trabajo está hecho en codespaces de github, por lo que a la hora de abrir el URL 
// "http://localhost:8080/cierre", no lo encuentra, ya que github actúa en una nube, da su propia URL, 
// sin embargo, al abrir la que me da al ejecutar el codigo, no funciona si no se agrega al final 
// de la URL "/cierre"